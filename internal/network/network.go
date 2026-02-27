package network

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf16"

	"cleanforge/internal/cmd"
	"golang.org/x/sys/windows/registry"
)

// Adapter cache — set by GetNetworkStatus(), reused by SetDNS()/ResetDNS()
// to avoid redundant slow PowerShell calls.
var (
	cachedAdapter   string
	cachedAdapterMu sync.RWMutex
)

// cmdTimeout is the default timeout for PowerShell/netsh commands.
const cmdTimeout = 10 * time.Second

// elevatedTimeout is a longer timeout for commands that trigger UAC prompts.
const elevatedTimeout = 30 * time.Second

// encodePSCommand encodes a PowerShell command as UTF-16LE Base64 for use with
// -EncodedCommand. This avoids all quoting/escaping issues with nested commands.
func encodePSCommand(psCommand string) string {
	runes := utf16.Encode([]rune(psCommand))
	b := make([]byte, len(runes)*2)
	for i, r := range runes {
		b[i*2] = byte(r)
		b[i*2+1] = byte(r >> 8)
	}
	return base64.StdEncoding.EncodeToString(b)
}

// runElevated runs a PowerShell command with UAC elevation (Start-Process -Verb RunAs).
// Uses -EncodedCommand to avoid quoting issues. Triggers a UAC prompt for the user.
func runElevated(ctx context.Context, psCommand string) ([]byte, error) {
	encoded := encodePSCommand(psCommand)
	elevateCmd := fmt.Sprintf(
		`Start-Process powershell -Verb RunAs -Wait -WindowStyle Hidden -ArgumentList '-NoProfile -EncodedCommand %s'`,
		encoded,
	)
	return cmd.HiddenContext(ctx, "powershell", "-NoProfile", "-Command", elevateCmd).CombinedOutput()
}

func getCachedAdapter() string {
	cachedAdapterMu.RLock()
	defer cachedAdapterMu.RUnlock()
	return cachedAdapter
}

func setCachedAdapter(name string) {
	cachedAdapterMu.Lock()
	defer cachedAdapterMu.Unlock()
	cachedAdapter = name
}

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
// Uses PowerShell as primary method (locale-independent), with netsh as fallback.
// Never returns an error — always returns at least partial status so the UI can render.
// Caches the adapter name for use by SetDNS/ResetDNS to avoid redundant slow calls.
func GetNetworkStatus() (*NetworkStatus, error) {
	status := &NetworkStatus{}

	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()

	// Primary: Single PowerShell call to get ALL network info (locale-independent)
	psCmd := `$cfg = Get-NetIPConfiguration | Where-Object { $_.IPv4DefaultGateway -ne $null } | Select-Object -First 1
if ($cfg) {
    $dns = (Get-DnsClientServerAddress -InterfaceIndex $cfg.InterfaceIndex -AddressFamily IPv4 -ErrorAction SilentlyContinue).ServerAddresses -join ', '
    "$($cfg.InterfaceAlias)|$($cfg.IPv4Address.IPAddress)|$($cfg.IPv4DefaultGateway.NextHop)|$dns"
}`
	if out, err := cmd.HiddenContext(ctx, "powershell", "-NoProfile", "-Command", psCmd).Output(); err == nil {
		result := strings.TrimSpace(string(out))
		if result != "" {
			parts := strings.SplitN(result, "|", 4)
			if len(parts) >= 1 && strings.TrimSpace(parts[0]) != "" {
				status.Adapter = strings.TrimSpace(parts[0])
			}
			if len(parts) >= 2 && strings.TrimSpace(parts[1]) != "" {
				status.IPAddress = strings.TrimSpace(parts[1])
			}
			if len(parts) >= 3 && strings.TrimSpace(parts[2]) != "" {
				status.Gateway = strings.TrimSpace(parts[2])
			}
			if len(parts) >= 4 && strings.TrimSpace(parts[3]) != "" {
				status.CurrentDNS = strings.TrimSpace(parts[3])
			}
		}
	}

	// Fallback 1: Try getting adapter via GetActiveAdapter if PowerShell didn't find one
	if status.Adapter == "" {
		if adapter, err := GetActiveAdapter(); err == nil {
			status.Adapter = adapter
		}
	}

	// Fallback 2: netsh parsing with multi-locale support if still missing data
	if status.Adapter != "" && (status.IPAddress == "" || status.CurrentDNS == "") {
		ctx2, cancel2 := context.WithTimeout(context.Background(), cmdTimeout)
		defer cancel2()
		if out, err := cmd.HiddenContext(ctx2, "netsh", "interface", "ip", "show", "config", "name="+status.Adapter).CombinedOutput(); err == nil {
			lines := strings.Split(string(out), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)

				// Look for any line with an IP address after a colon separator
				if strings.Contains(line, ":") {
					colParts := strings.SplitN(line, ":", 2)
					if len(colParts) == 2 {
						val := strings.TrimSpace(colParts[1])
						label := strings.ToLower(colParts[0])

						if net.ParseIP(val) != nil {
							// IP Address (EN: "IP Address", PT: "Endereço IP")
							if status.IPAddress == "" && (strings.Contains(label, "ip") && !strings.Contains(label, "dns") && !strings.Contains(label, "gateway")) {
								status.IPAddress = val
							}
							// DNS (EN: "DNS Servers", PT: "Servidores DNS")
							if status.CurrentDNS == "" && strings.Contains(label, "dns") {
								status.CurrentDNS = val
							}
						}

						// Gateway
						if status.Gateway == "" && strings.Contains(label, "gateway") && val != "" {
							status.Gateway = val
						}
					}
				}
			}
		}
	}

	// Fallback 3: If we still don't have DNS, check via PowerShell directly
	if status.CurrentDNS == "" && status.Adapter != "" {
		ctx3, cancel3 := context.WithTimeout(context.Background(), cmdTimeout)
		defer cancel3()
		psDNS := fmt.Sprintf(`(Get-DnsClientServerAddress -InterfaceAlias '%s' -AddressFamily IPv4 -ErrorAction SilentlyContinue).ServerAddresses -join ', '`, status.Adapter)
		if out, err := cmd.HiddenContext(ctx3, "powershell", "-NoProfile", "-Command", psDNS).Output(); err == nil {
			dns := strings.TrimSpace(string(out))
			if dns != "" {
				status.CurrentDNS = dns
			}
		}
	}

	// Cache the adapter name for SetDNS/ResetDNS to reuse
	if status.Adapter != "" {
		setCachedAdapter(status.Adapter)
	}

	// Check Nagle status
	status.NagleDisabled = isNagleDisabled()

	return status, nil
}

// getAdapter returns the cached adapter name or fetches it fresh.
// Uses the cache from GetNetworkStatus() to avoid slow PowerShell re-calls.
func getAdapter() (string, error) {
	// Try cache first (set by GetNetworkStatus)
	if cached := getCachedAdapter(); cached != "" {
		return cached, nil
	}
	// Fall back to fresh detection
	adapter, err := GetActiveAdapter()
	if err != nil {
		return "", err
	}
	setCachedAdapter(adapter)
	return adapter, nil
}

// SetDNS applies the given DNS preset to the active network adapter.
// Tries non-elevated PowerShell first; if it needs admin, triggers UAC elevation.
func SetDNS(preset DNSPreset) error {
	adapter, err := getAdapter()
	if err != nil {
		return fmt.Errorf("no active network adapter found: %w", err)
	}

	dnsCmd := fmt.Sprintf(
		`Set-DnsClientServerAddress -InterfaceAlias '%s' -ServerAddresses @('%s','%s')`,
		adapter, preset.Primary, preset.Secondary,
	)

	// Try non-elevated first
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()
	if _, psErr := cmd.HiddenContext(ctx, "powershell", "-NoProfile", "-Command", dnsCmd).CombinedOutput(); psErr == nil {
		return nil
	}

	// Needs elevation — trigger UAC prompt
	ctx2, cancel2 := context.WithTimeout(context.Background(), elevatedTimeout)
	defer cancel2()
	out, err := runElevated(ctx2, dnsCmd)
	if err != nil {
		return fmt.Errorf("failed to set DNS: %s - %w", strings.TrimSpace(string(out)), err)
	}

	return nil
}

// ResetDNS resets the DNS configuration to DHCP (automatic) for the active adapter.
// Tries non-elevated PowerShell first; if it needs admin, triggers UAC elevation.
func ResetDNS() error {
	adapter, err := getAdapter()
	if err != nil {
		return fmt.Errorf("no active network adapter found: %w", err)
	}

	dnsCmd := fmt.Sprintf(`Set-DnsClientServerAddress -InterfaceAlias '%s' -ResetServerAddresses`, adapter)

	// Try non-elevated first
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()
	if _, psErr := cmd.HiddenContext(ctx, "powershell", "-NoProfile", "-Command", dnsCmd).CombinedOutput(); psErr == nil {
		return nil
	}

	// Needs elevation — trigger UAC prompt
	ctx2, cancel2 := context.WithTimeout(context.Background(), elevatedTimeout)
	defer cancel2()
	out, err := runElevated(ctx2, dnsCmd)
	if err != nil {
		return fmt.Errorf("failed to reset DNS to DHCP: %s - %w", strings.TrimSpace(string(out)), err)
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

// GetActiveAdapter finds the currently active network adapter name.
// Uses PowerShell as primary method (locale-independent), with netsh as fallback.
func GetActiveAdapter() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()

	// Primary: PowerShell (works on any Windows locale)
	psCmd := `(Get-NetIPConfiguration | Where-Object { $_.IPv4DefaultGateway -ne $null } | Select-Object -First 1).InterfaceAlias`
	if out, err := cmd.HiddenContext(ctx, "powershell", "-NoProfile", "-Command", psCmd).Output(); err == nil {
		adapter := strings.TrimSpace(string(out))
		if adapter != "" {
			return adapter, nil
		}
	}

	// Fallback 1: netsh with locale-tolerant regex
	// Section headers across all locales have the adapter name in quotes:
	//   EN: Configuration for interface "Wi-Fi"
	//   PT: Configuração da interface "Wi-Fi"
	//   ES: Configuración de la interfaz "Wi-Fi"
	out, err := cmd.Hidden("netsh", "interface", "ip", "show", "config").CombinedOutput()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		configRe := regexp.MustCompile(`"([^"]+)"`)
		ipRe := regexp.MustCompile(`(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})`)

		var currentAdapter string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)

			// Section headers are not indented and contain a quoted adapter name
			if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
				if matches := configRe.FindStringSubmatch(trimmed); len(matches) > 1 {
					currentAdapter = matches[1]
					continue
				}
			}

			// Look for IP addresses under the current adapter
			if currentAdapter != "" {
				if matches := ipRe.FindStringSubmatch(trimmed); len(matches) > 1 {
					ip := matches[1]
					parsed := net.ParseIP(ip)
					if parsed != nil && !parsed.IsLoopback() && ip != "0.0.0.0" && !strings.HasPrefix(ip, "169.254.") {
						return currentAdapter, nil
					}
				}
			}
		}
	}

	// Fallback 2: route + ipconfig
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

	// Try to match this IP to an adapter name via PowerShell (locale-independent)
	psAdapter := fmt.Sprintf(`(Get-NetIPAddress -IPAddress '%s' -ErrorAction SilentlyContinue | Get-NetAdapter -ErrorAction SilentlyContinue).Name`, defaultIP)
	if out, err := cmd.Hidden("powershell", "-NoProfile", "-Command", psAdapter).Output(); err == nil {
		adapter := strings.TrimSpace(string(out))
		if adapter != "" {
			return adapter, nil
		}
	}

	// Last resort: ipconfig parsing with locale-tolerant patterns
	ipconfigOut, err := cmd.Hidden("ipconfig").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to run ipconfig: %w", err)
	}

	ipconfigLines := strings.Split(string(ipconfigOut), "\n")
	// Adapter headers in ipconfig end with ":" and are not indented
	// EN: "Wireless LAN adapter Wi-Fi:" / PT: "Adaptador de Rede sem Fio Wi-Fi:"
	ipLineRe := regexp.MustCompile(`(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})`)

	var adapterName string
	for _, line := range ipconfigLines {
		trimmed := strings.TrimSpace(line)
		// Adapter headers: not indented, end with ":"
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && strings.HasSuffix(trimmed, ":") && trimmed != ":" {
			// Extract just the last word before ":" as a best-effort adapter name
			adapterName = strings.TrimSuffix(trimmed, ":")
			// Try to extract just the adapter alias (after the last space in the type prefix)
			// This works because adapter names like "Wi-Fi" or "Ethernet" are the suffix
			continue
		}
		if adapterName != "" {
			if matches := ipLineRe.FindStringSubmatch(trimmed); len(matches) > 1 {
				if matches[1] == defaultIP {
					// The adapterName from ipconfig is the full description, but for netsh
					// we need the interface alias. Try to extract it from the line.
					// On both EN and PT, the interface alias is the last part after the type.
					// However, this is unreliable. Since we have the IP, use PowerShell to get the alias.
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
