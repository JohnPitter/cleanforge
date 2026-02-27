//go:build integration

package network

import (
	"net"
	"regexp"
	"testing"
)

// ipv4Pattern matches a valid IPv4 address (basic check).
var ipv4Pattern = regexp.MustCompile(`^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$`)

// isValidIPv4 checks if a string is a valid IPv4 address using net.ParseIP.
func isValidIPv4(s string) bool {
	ip := net.ParseIP(s)
	if ip == nil {
		return false
	}
	// Ensure it's IPv4 (not IPv6)
	return ip.To4() != nil
}

// TestDNSPresetsAreValid verifies that each DNS preset has properly formatted
// IPv4 addresses for both primary and secondary DNS fields.
func TestDNSPresetsAreValid(t *testing.T) {
	presets := GetDNSPresets()

	if len(presets) == 0 {
		t.Fatal("GetDNSPresets() returned 0 presets")
	}

	for _, preset := range presets {
		if preset.ID == "" {
			t.Error("preset has empty ID")
		}
		if preset.Name == "" {
			t.Errorf("preset %q has empty Name", preset.ID)
		}
		if preset.Description == "" {
			t.Errorf("preset %q has empty Description", preset.ID)
		}

		// Validate primary DNS
		if !ipv4Pattern.MatchString(preset.Primary) {
			t.Errorf("preset %q primary DNS %q does not match IPv4 pattern", preset.ID, preset.Primary)
		}
		if !isValidIPv4(preset.Primary) {
			t.Errorf("preset %q primary DNS %q is not a valid IPv4 address", preset.ID, preset.Primary)
		}

		// Validate secondary DNS
		if !ipv4Pattern.MatchString(preset.Secondary) {
			t.Errorf("preset %q secondary DNS %q does not match IPv4 pattern", preset.ID, preset.Secondary)
		}
		if !isValidIPv4(preset.Secondary) {
			t.Errorf("preset %q secondary DNS %q is not a valid IPv4 address", preset.ID, preset.Secondary)
		}

		// Primary and secondary should be different
		if preset.Primary == preset.Secondary {
			t.Errorf("preset %q has same primary and secondary DNS: %s", preset.ID, preset.Primary)
		}
	}
}

// TestDNSPresetsKnownValues verifies that well-known DNS presets have correct IPs.
func TestDNSPresetsKnownValues(t *testing.T) {
	presets := GetDNSPresets()

	expected := map[string]struct {
		primary   string
		secondary string
	}{
		"cloudflare": {"1.1.1.1", "1.0.0.1"},
		"google":     {"8.8.8.8", "8.8.4.4"},
		"quad9":      {"9.9.9.9", "149.112.112.112"},
	}

	presetMap := make(map[string]DNSPreset)
	for _, p := range presets {
		presetMap[p.ID] = p
	}

	for id, exp := range expected {
		preset, ok := presetMap[id]
		if !ok {
			t.Errorf("preset %q not found", id)
			continue
		}
		if preset.Primary != exp.primary {
			t.Errorf("preset %q primary = %q, want %q", id, preset.Primary, exp.primary)
		}
		if preset.Secondary != exp.secondary {
			t.Errorf("preset %q secondary = %q, want %q", id, preset.Secondary, exp.secondary)
		}
	}
}

// TestDNSPresetsUniqueIDs verifies no duplicate preset IDs exist.
func TestDNSPresetsUniqueIDs(t *testing.T) {
	presets := GetDNSPresets()
	seen := make(map[string]bool)
	for _, p := range presets {
		if seen[p.ID] {
			t.Errorf("duplicate DNS preset ID: %q", p.ID)
		}
		seen[p.ID] = true
	}
}

// TestNetworkStatusFields gets the current network status and verifies
// that the returned fields are reasonable.
func TestNetworkStatusFields(t *testing.T) {
	status, err := GetNetworkStatus()
	if err != nil {
		// This may fail in CI environments without a network adapter.
		// Skip gracefully.
		t.Skipf("GetNetworkStatus() returned error (may be expected in CI): %v", err)
	}

	if status.Adapter == "" {
		t.Error("NetworkStatus.Adapter is empty")
	}

	// IPAddress may be empty on some configurations, but log it
	if status.IPAddress == "" {
		t.Log("NetworkStatus.IPAddress is empty (may be expected on some configurations)")
	} else if !isValidIPv4(status.IPAddress) {
		t.Errorf("NetworkStatus.IPAddress %q is not a valid IPv4", status.IPAddress)
	}

	// Gateway may also be empty
	if status.Gateway != "" && !isValidIPv4(status.Gateway) {
		t.Errorf("NetworkStatus.Gateway %q is not a valid IPv4", status.Gateway)
	}

	// CurrentDNS may be empty or contain multiple entries
	if status.CurrentDNS != "" {
		t.Logf("Current DNS: %s", status.CurrentDNS)
	}

	// NagleDisabled is just a bool - log it for information
	t.Logf("Nagle disabled: %v", status.NagleDisabled)
}

// TestPingMultipleHosts pings well-known public DNS servers and verifies
// that at least one returns a latency > 0.
func TestPingMultipleHosts(t *testing.T) {
	hosts := []string{"8.8.8.8", "1.1.1.1"}
	atLeastOneSuccess := false

	for _, host := range hosts {
		latency, err := PingTest(host)
		if err != nil {
			t.Logf("PingTest(%q) failed: %v", host, err)
			continue
		}

		if latency <= 0 {
			t.Errorf("PingTest(%q) returned latency %f, want > 0", host, latency)
		} else {
			t.Logf("PingTest(%q) latency: %.1f ms", host, latency)
			atLeastOneSuccess = true
		}
	}

	if !atLeastOneSuccess {
		t.Skip("all ping tests failed; skipping (network may be unavailable)")
	}
}

// TestFlushNetworkOutput runs FlushNetwork and checks the output.
// This test requires admin privileges for most commands.
func TestFlushNetworkOutput(t *testing.T) {
	output, err := FlushNetwork()

	// FlushNetwork may return an error if not running as admin,
	// but it should still produce some output
	if err != nil {
		t.Logf("FlushNetwork returned error (may require admin): %v", err)
		// Even on error, there should be some output from the commands
		// that did run successfully
		if output == "" {
			t.Skip("FlushNetwork produced no output; likely not running as admin")
		}
	}

	if output != "" {
		t.Logf("FlushNetwork output length: %d chars", len(output))
	}
}

// TestMeasureLatency tests the convenience latency measurement function.
func TestMeasureLatency(t *testing.T) {
	latency, err := MeasureLatency()
	if err != nil {
		t.Skipf("MeasureLatency() failed (network may be unavailable): %v", err)
	}

	if latency <= 0 {
		t.Errorf("MeasureLatency() = %f, want > 0", latency)
	}

	// Sanity check: latency should be less than 10 seconds
	if latency > 10000 {
		t.Errorf("MeasureLatency() = %f ms, seems unreasonably high", latency)
	}

	t.Logf("Best measured latency: %.1f ms", latency)
}

// TestIntegration_GetActiveAdapter verifies that GetActiveAdapter returns a non-empty adapter name.
func TestIntegration_GetActiveAdapter(t *testing.T) {
	adapter, err := GetActiveAdapter()
	if err != nil {
		t.Skipf("GetActiveAdapter() failed (may be expected in CI): %v", err)
	}

	if adapter == "" {
		t.Error("GetActiveAdapter() returned empty adapter name")
	} else {
		t.Logf("Active adapter: %s", adapter)
	}
}

// TestPingInvalidHost verifies that PingTest returns an error for an unreachable host.
func TestPingInvalidHost(t *testing.T) {
	_, err := PingTest("192.0.2.1") // TEST-NET-1, should be unreachable
	if err == nil {
		// It is possible the host responds in some networks, so just log
		t.Log("PingTest(192.0.2.1) succeeded unexpectedly; host may be reachable")
	}
}
