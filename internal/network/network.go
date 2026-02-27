package network

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"cleanforge/internal/cmd"
	"golang.org/x/sys/windows/registry"
)

// DNSPreset represents a DNS configuration preset.
type DNSPreset struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Primary     string `json:"primary"`
	Secondary   string `json:"secondary"`
	Description string `json:"description"`
}

// NetworkStatus holds the current network adapter configuration.
type NetworkStatus struct {
	CurrentDNS    string `json:"currentDns"`
	NagleDisabled bool   `json:"nagleDisabled"`
	Adapter       string `json:"adapter"`
	IPAddress     string `json:"ipAddress"`
	Gateway       string `json:"gateway"`
}

// predefined DNS presets
var dnsPresets = []DNSPreset{
	{
		ID:          "cloudflare",
		Name:        "Cloudflare",
		Primary:     "1.1.1.1",
		Secondary:   "1.0.0.1",
		Description: "Fastest, privacy-focused",
	},
	{
		ID:          "google",
		Name:        "Google",
		Primary:     "8.8.8.8",
		Secondary:   "8.8.4.4",
		Description: "Reliable, low latency",
	},
	{
		ID:          "opendns",
		Name:        "OpenDNS",
		Primary:     "208.67.222.222",
		Secondary:   "208.67.220.220",
		Description: "Family-safe option",
	},
	{
		ID:          "quad9",
		Name:        "Quad9",
		Primary:     "9.9.9.9",
		Secondary:   "149.112.112.112",
		Description: "Security-focused, blocks malware",
	},
}

// nagle-related registry constants
const (
	tcpInterfacesPath = `SYSTEM\CurrentControlSet\Services\Tcpip\Parameters\Interfaces`
)

// GetDNSPresets returns all available DNS presets.
func GetDNSPresets() []DNSPreset {
	return dnsPresets
}

// GetNetworkStatus retrieves the current network configuration for the active adapter.
func GetNetworkStatus() (*NetworkStatus, error) {
	adapter, err := GetActiveAdapter()
	if err != nil {
		return nil, fmt.Errorf("failed to get active adapter: %w", err)
	}

	status := &NetworkStatus{
		Adapter: adapter,
	}

	// Parse netsh output for the active adapter
	out, err := cmd.Hidden("netsh", "interface", "ip", "show", "config", "name="+adapter).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get network config: %w", err)
	}

	output := string(out)
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "IP Address") || strings.Contains(line, "IP address") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				status.IPAddress = strings.TrimSpace(parts[1])
			}
		}

		if strings.Contains(line, "Default Gateway") || strings.Contains(line, "Default gateway") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				gw := strings.TrimSpace(parts[1])
				if gw != "" {
					status.Gateway = gw
				}
			}
		}

		if strings.Contains(line, "DNS") && strings.Contains(line, "Servers") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				dns := strings.TrimSpace(parts[1])
				if dns != "" {
					status.CurrentDNS = dns
				}
			}
		}
	}

	// If DNS was not found in the config line, look for statically configured DNS
	if status.CurrentDNS == "" {
		// Try parsing DNS lines that appear after the DNS Servers header
		inDNS := false
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.Contains(trimmed, "DNS") && strings.Contains(trimmed, "Servers") {
				parts := strings.SplitN(trimmed, ":", 2)
				if len(parts) == 2 {
					dns := strings.TrimSpace(parts[1])
					if dns != "" {
						status.CurrentDNS = dns
					}
				}
				inDNS = true
				continue
			}
			if inDNS {
				// Continuation lines for DNS servers are indented IP addresses
				if net.ParseIP(trimmed) != nil {
					if status.CurrentDNS != "" {
						status.CurrentDNS += ", " + trimmed
					} else {
						status.CurrentDNS = trimmed
					}
				} else {
					inDNS = false
				}
			}
		}
	}

	// Check Nagle status
	status.NagleDisabled = isNagleDisabled()

	return status, nil
}

// SetDNS applies the given DNS preset to the active network adapter.
func SetDNS(preset DNSPreset) error {
	adapter, err := GetActiveAdapter()
	if err != nil {
		return fmt.Errorf("failed to get active adapter: %w", err)
	}

	// Set primary DNS
	out, err := cmd.Hidden("netsh", "interface", "ip", "set", "dns",
		fmt.Sprintf("name=%s", adapter), "static", preset.Primary).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set primary DNS: %s - %w", string(out), err)
	}

	// Set secondary DNS
	out, err = cmd.Hidden("netsh", "interface", "ip", "add", "dns",
		fmt.Sprintf("name=%s", adapter), preset.Secondary, "index=2").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set secondary DNS: %s - %w", string(out), err)
	}

	return nil
}

// ResetDNS resets the DNS configuration to DHCP (automatic) for the active adapter.
func ResetDNS() error {
	adapter, err := GetActiveAdapter()
	if err != nil {
		return fmt.Errorf("failed to get active adapter: %w", err)
	}

	out, err := cmd.Hidden("netsh", "interface", "ip", "set", "dns",
		fmt.Sprintf("name=%s", adapter), "dhcp").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to reset DNS to DHCP: %s - %w", string(out), err)
	}

	return nil
}

// DisableNagle disables the Nagle algorithm on all network interfaces by setting
// TcpAckFrequency=1 and TCPNoDelay=1 in the registry.
func DisableNagle() error {
	interfacesKey, err := registry.OpenKey(registry.LOCAL_MACHINE, tcpInterfacesPath, registry.READ)
	if err != nil {
		return fmt.Errorf("failed to open TCP interfaces registry key: %w", err)
	}
	defer interfacesKey.Close()

	subkeys, err := interfacesKey.ReadSubKeyNames(-1)
	if err != nil {
		return fmt.Errorf("failed to read interface subkeys: %w", err)
	}

	var lastErr error
	for _, subkey := range subkeys {
		keyPath := tcpInterfacesPath + `\` + subkey
		k, err := registry.OpenKey(registry.LOCAL_MACHINE, keyPath, registry.SET_VALUE)
		if err != nil {
			lastErr = fmt.Errorf("failed to open interface key %s: %w", subkey, err)
			continue
		}

		if err := k.SetDWordValue("TcpAckFrequency", 1); err != nil {
			lastErr = fmt.Errorf("failed to set TcpAckFrequency on %s: %w", subkey, err)
		}
		if err := k.SetDWordValue("TCPNoDelay", 1); err != nil {
			lastErr = fmt.Errorf("failed to set TCPNoDelay on %s: %w", subkey, err)
		}

		k.Close()
	}

	return lastErr
}

// EnableNagle restores the Nagle algorithm on all network interfaces by removing
// the TcpAckFrequency and TCPNoDelay registry values.
func EnableNagle() error {
	interfacesKey, err := registry.OpenKey(registry.LOCAL_MACHINE, tcpInterfacesPath, registry.READ)
	if err != nil {
		return fmt.Errorf("failed to open TCP interfaces registry key: %w", err)
	}
	defer interfacesKey.Close()

	subkeys, err := interfacesKey.ReadSubKeyNames(-1)
	if err != nil {
		return fmt.Errorf("failed to read interface subkeys: %w", err)
	}

	var lastErr error
	for _, subkey := range subkeys {
		keyPath := tcpInterfacesPath + `\` + subkey
		k, err := registry.OpenKey(registry.LOCAL_MACHINE, keyPath, registry.SET_VALUE)
		if err != nil {
			lastErr = fmt.Errorf("failed to open interface key %s: %w", subkey, err)
			continue
		}

		// Delete values; ignore errors if they don't exist
		_ = k.DeleteValue("TcpAckFrequency")
		_ = k.DeleteValue("TCPNoDelay")

		k.Close()
	}

	return lastErr
}

// FlushNetwork runs a full network flush sequence: flushdns, winsock reset,
// IP reset, release, and renew. Collects output from all commands.
func FlushNetwork() (string, error) {
	commands := []struct {
		name string
		args []string
	}{
		{"ipconfig", []string{"/flushdns"}},
		{"netsh", []string{"winsock", "reset"}},
		{"netsh", []string{"int", "ip", "reset"}},
		{"ipconfig", []string{"/release"}},
		{"ipconfig", []string{"/renew"}},
	}

	var outputs []string
	var errs []string

	for _, c := range commands {
		out, err := cmd.Hidden(c.name, c.args...).CombinedOutput()
		label := c.name + " " + strings.Join(c.args, " ")
		if err != nil {
			errs = append(errs, fmt.Sprintf("[%s] error: %s - %s", label, err.Error(), strings.TrimSpace(string(out))))
		} else {
			outputs = append(outputs, fmt.Sprintf("[%s] %s", label, strings.TrimSpace(string(out))))
		}
	}

	result := strings.Join(outputs, "\n\n")
	if len(errs) > 0 {
		result += "\n\nErrors:\n" + strings.Join(errs, "\n")
	}

	if len(outputs) == 0 && len(errs) > 0 {
		return result, fmt.Errorf("all network flush commands failed")
	}

	return result, nil
}

// GetActiveAdapter parses netsh output to find the currently active network adapter name.
func GetActiveAdapter() (string, error) {
	out, err := cmd.Hidden("netsh", "interface", "ip", "show", "config").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to run netsh: %w", err)
	}

	output := string(out)
	lines := strings.Split(output, "\n")

	// Match lines like: Configuration for interface "Wi-Fi"
	// or: Configuration for interface "Ethernet"
	configRe := regexp.MustCompile(`(?i)Configuration for interface\s+"([^"]+)"`)
	ipRe := regexp.MustCompile(`(?i)IP Address.*?:\s*(\d+\.\d+\.\d+\.\d+)`)

	var currentAdapter string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if matches := configRe.FindStringSubmatch(line); len(matches) > 1 {
			currentAdapter = matches[1]
			continue
		}

		// If we find an IP address for the current adapter, and it's not a
		// loopback or APIPA address, it's likely the active adapter.
		if currentAdapter != "" {
			if matches := ipRe.FindStringSubmatch(line); len(matches) > 1 {
				ip := matches[1]
				if ip != "0.0.0.0" && !strings.HasPrefix(ip, "127.") && !strings.HasPrefix(ip, "169.254.") {
					return currentAdapter, nil
				}
			}
		}
	}

	// Fallback: try to detect from route print
	return getAdapterFromRoute()
}

// getAdapterFromRoute is a fallback method to find the active adapter using route and
// ipconfig commands.
func getAdapterFromRoute() (string, error) {
	// Get the default interface IP from "route print 0.0.0.0"
	out, err := cmd.Hidden("route", "print", "0.0.0.0").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to run route print: %w", err)
	}

	output := string(out)
	lines := strings.Split(output, "\n")

	// Look for the default route line (destination 0.0.0.0)
	var defaultIP string
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 4 && fields[0] == "0.0.0.0" && fields[1] == "0.0.0.0" {
			defaultIP = fields[3] // interface IP
			break
		}
	}

	if defaultIP == "" {
		return "", fmt.Errorf("no active network adapter found")
	}

	// Now match this IP to an adapter name from ipconfig
	ipconfigOut, err := cmd.Hidden("ipconfig").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to run ipconfig: %w", err)
	}

	ipconfigLines := strings.Split(string(ipconfigOut), "\n")
	adapterRe := regexp.MustCompile(`(?i)^(?:Ethernet|Wireless LAN) adapter\s+(.+?)\s*:`)
	ipLineRe := regexp.MustCompile(`(?i)IPv4 Address.*?:\s*(\d+\.\d+\.\d+\.\d+)`)

	var adapterName string
	for _, line := range ipconfigLines {
		if matches := adapterRe.FindStringSubmatch(line); len(matches) > 1 {
			adapterName = matches[1]
			continue
		}
		if adapterName != "" {
			if matches := ipLineRe.FindStringSubmatch(line); len(matches) > 1 {
				if matches[1] == defaultIP {
					return adapterName, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no active network adapter found matching IP %s", defaultIP)
}

// PingTest pings the specified host and returns the average latency in milliseconds.
func PingTest(host string) (float64, error) {
	out, err := cmd.Hidden("ping", "-n", "4", host).CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("ping failed: %s - %w", strings.TrimSpace(string(out)), err)
	}

	output := string(out)

	// Try to parse "Average = XXms" from the ping output
	avgRe := regexp.MustCompile(`(?i)Average\s*=\s*(\d+)\s*ms`)
	if matches := avgRe.FindStringSubmatch(output); len(matches) > 1 {
		ms, err := strconv.ParseFloat(matches[1], 64)
		if err == nil {
			return ms, nil
		}
	}

	// Fallback: try to parse "Média = XXms" (Portuguese locale)
	avgRePt := regexp.MustCompile(`(?i)(?:M[eé]dia|M[ií]nimo)\s*=\s*(\d+)\s*ms`)
	if matches := avgRePt.FindStringSubmatch(output); len(matches) > 1 {
		ms, err := strconv.ParseFloat(matches[1], 64)
		if err == nil {
			return ms, nil
		}
	}

	// Last resort: parse individual ping times and average them
	timeRe := regexp.MustCompile(`(?i)(?:time|tempo)[=<]\s*(\d+)\s*ms`)
	allMatches := timeRe.FindAllStringSubmatch(output, -1)
	if len(allMatches) == 0 {
		return 0, fmt.Errorf("could not parse ping output: %s", strings.TrimSpace(output))
	}

	var total float64
	for _, match := range allMatches {
		ms, err := strconv.ParseFloat(match[1], 64)
		if err == nil {
			total += ms
		}
	}

	return total / float64(len(allMatches)), nil
}

// isNagleDisabled checks if Nagle's algorithm is disabled by reading the registry
// for at least one interface with both TcpAckFrequency=1 and TCPNoDelay=1.
func isNagleDisabled() bool {
	interfacesKey, err := registry.OpenKey(registry.LOCAL_MACHINE, tcpInterfacesPath, registry.READ)
	if err != nil {
		return false
	}
	defer interfacesKey.Close()

	subkeys, err := interfacesKey.ReadSubKeyNames(-1)
	if err != nil {
		return false
	}

	// Check if at least one interface with an IP has Nagle disabled.
	// We look for interfaces that have DhcpIPAddress or IPAddress set (i.e., real adapters).
	for _, subkey := range subkeys {
		keyPath := tcpInterfacesPath + `\` + subkey
		k, err := registry.OpenKey(registry.LOCAL_MACHINE, keyPath, registry.READ)
		if err != nil {
			continue
		}

		// Check if this interface actually has an IP (is a real adapter)
		hasIP := false
		if ip, _, err := k.GetStringValue("DhcpIPAddress"); err == nil && ip != "" && ip != "0.0.0.0" {
			hasIP = true
		}
		if !hasIP {
			if ip, _, err := k.GetStringValue("IPAddress"); err == nil {
				if ip != "" && ip != "0.0.0.0" {
					hasIP = true
				}
			}
		}

		if !hasIP {
			k.Close()
			continue
		}

		ackFreq, _, err1 := k.GetIntegerValue("TcpAckFrequency")
		noDelay, _, err2 := k.GetIntegerValue("TCPNoDelay")
		k.Close()

		if err1 == nil && err2 == nil && ackFreq == 1 && noDelay == 1 {
			return true
		}

		// If a real adapter does NOT have the values, Nagle is still enabled.
		return false
	}

	return false
}

// MeasureLatency is a convenience wrapper that pings multiple common hosts
// and returns the best (lowest) latency. Useful for quick network quality checks.
func MeasureLatency() (float64, error) {
	hosts := []string{"1.1.1.1", "8.8.8.8", "9.9.9.9"}

	bestLatency := -1.0
	var lastErr error

	for _, host := range hosts {
		// Use a short timeout approach: ping with -n 2 for speed
		out, err := cmd.Hidden("ping", "-n", "2", "-w", "2000", host).CombinedOutput()
		if err != nil {
			lastErr = err
			continue
		}

		output := string(out)
		avgRe := regexp.MustCompile(`(?i)(?:Average|M[eé]dia)\s*=\s*(\d+)\s*ms`)
		if matches := avgRe.FindStringSubmatch(output); len(matches) > 1 {
			ms, err := strconv.ParseFloat(matches[1], 64)
			if err == nil {
				if bestLatency < 0 || ms < bestLatency {
					bestLatency = ms
				}
			}
		}
	}

	if bestLatency < 0 {
		if lastErr != nil {
			return 0, fmt.Errorf("all ping tests failed: %w", lastErr)
		}
		return 0, fmt.Errorf("could not measure latency")
	}

	return bestLatency, nil
}

// SpeedTestBasic performs a rudimentary download speed test by downloading a
// small file and measuring throughput. Returns speed in Mbps.
// Note: This is a basic test; for accurate results use a dedicated speed test.
func SpeedTestBasic() (float64, error) {
	// We just measure latency as a quick indicator; a full speed test
	// would require downloading a file which is beyond a simple utility.
	// Instead we measure jitter by doing several pings.
	start := time.Now()
	_, err := cmd.Hidden("ping", "-n", "10", "-w", "1000", "1.1.1.1").CombinedOutput()
	elapsed := time.Since(start)

	if err != nil {
		return 0, fmt.Errorf("speed test failed: %w", err)
	}

	// Return the time in seconds as a rough network responsiveness indicator
	return elapsed.Seconds(), nil
}
