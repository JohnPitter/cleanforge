package network

import (
	"testing"
)

func TestGetDNSPresets(t *testing.T) {
	presets := GetDNSPresets()

	if len(presets) != 4 {
		t.Fatalf("expected 4 DNS presets, got %d", len(presets))
	}

	expectedIDs := map[string]bool{
		"cloudflare": true,
		"google":     true,
		"opendns":    true,
		"quad9":      true,
	}

	for _, p := range presets {
		if !expectedIDs[p.ID] {
			t.Errorf("unexpected DNS preset ID: %q", p.ID)
		}
		if p.Name == "" {
			t.Errorf("preset %q has empty Name", p.ID)
		}
		if p.Primary == "" {
			t.Errorf("preset %q has empty Primary DNS", p.ID)
		}
		if p.Secondary == "" {
			t.Errorf("preset %q has empty Secondary DNS", p.ID)
		}
		if p.Description == "" {
			t.Errorf("preset %q has empty Description", p.ID)
		}
	}
}

func TestGetDNSPresetsContent(t *testing.T) {
	presets := GetDNSPresets()
	presetMap := make(map[string]DNSPreset)
	for _, p := range presets {
		presetMap[p.ID] = p
	}

	t.Run("Cloudflare", func(t *testing.T) {
		p := presetMap["cloudflare"]
		if p.Primary != "1.1.1.1" {
			t.Errorf("expected Cloudflare primary 1.1.1.1, got %q", p.Primary)
		}
		if p.Secondary != "1.0.0.1" {
			t.Errorf("expected Cloudflare secondary 1.0.0.1, got %q", p.Secondary)
		}
	})

	t.Run("Google", func(t *testing.T) {
		p := presetMap["google"]
		if p.Primary != "8.8.8.8" {
			t.Errorf("expected Google primary 8.8.8.8, got %q", p.Primary)
		}
		if p.Secondary != "8.8.4.4" {
			t.Errorf("expected Google secondary 8.8.4.4, got %q", p.Secondary)
		}
	})

	t.Run("Quad9", func(t *testing.T) {
		p := presetMap["quad9"]
		if p.Primary != "9.9.9.9" {
			t.Errorf("expected Quad9 primary 9.9.9.9, got %q", p.Primary)
		}
	})
}

func TestGetActiveAdapter(t *testing.T) {
	// This test verifies GetActiveAdapter does not panic.
	// It may fail on machines without a network connection.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("GetActiveAdapter panicked: %v", r)
		}
	}()

	adapter, err := GetActiveAdapter()
	if err != nil {
		t.Logf("GetActiveAdapter returned error (may be expected in some envs): %v", err)
		return
	}

	if adapter == "" {
		t.Error("GetActiveAdapter returned empty adapter name")
	}

	t.Logf("Active adapter: %q", adapter)
}

func TestGetNetworkStatus(t *testing.T) {
	// Non-panic test for GetNetworkStatus
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("GetNetworkStatus panicked: %v", r)
		}
	}()

	status, err := GetNetworkStatus()
	if err != nil {
		t.Logf("GetNetworkStatus returned error (may be expected): %v", err)
		return
	}

	if status == nil {
		t.Fatal("GetNetworkStatus returned nil without error")
	}

	if status.Adapter == "" {
		t.Error("NetworkStatus has empty Adapter")
	}

	t.Logf("Network: adapter=%q, ip=%q, dns=%q, nagle_disabled=%v",
		status.Adapter, status.IPAddress, status.CurrentDNS, status.NagleDisabled)
}

func TestPingTest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping ping test in short mode")
	}

	// Test with localhost which should always work
	latency, err := PingTest("127.0.0.1")
	if err != nil {
		t.Logf("PingTest(127.0.0.1) returned error: %v", err)
		return
	}

	if latency < 0 {
		t.Errorf("ping latency should not be negative, got %f", latency)
	}

	t.Logf("Ping to 127.0.0.1: %.2fms", latency)
}

func TestDNSPresetValues(t *testing.T) {
	presets := GetDNSPresets()
	presetMap := make(map[string]DNSPreset)
	for _, p := range presets {
		presetMap[p.ID] = p
	}

	t.Run("Cloudflare_is_1.1.1.1", func(t *testing.T) {
		cf, ok := presetMap["cloudflare"]
		if !ok {
			t.Fatal("cloudflare preset not found")
		}
		if cf.Primary != "1.1.1.1" {
			t.Errorf("Cloudflare primary: expected 1.1.1.1, got %q", cf.Primary)
		}
		if cf.Secondary != "1.0.0.1" {
			t.Errorf("Cloudflare secondary: expected 1.0.0.1, got %q", cf.Secondary)
		}
	})

	t.Run("Google_is_8.8.8.8", func(t *testing.T) {
		g, ok := presetMap["google"]
		if !ok {
			t.Fatal("google preset not found")
		}
		if g.Primary != "8.8.8.8" {
			t.Errorf("Google primary: expected 8.8.8.8, got %q", g.Primary)
		}
		if g.Secondary != "8.8.4.4" {
			t.Errorf("Google secondary: expected 8.8.4.4, got %q", g.Secondary)
		}
	})

	t.Run("OpenDNS_is_208.67.222.222", func(t *testing.T) {
		od, ok := presetMap["opendns"]
		if !ok {
			t.Fatal("opendns preset not found")
		}
		if od.Primary != "208.67.222.222" {
			t.Errorf("OpenDNS primary: expected 208.67.222.222, got %q", od.Primary)
		}
		if od.Secondary != "208.67.220.220" {
			t.Errorf("OpenDNS secondary: expected 208.67.220.220, got %q", od.Secondary)
		}
	})

	t.Run("Quad9_is_9.9.9.9", func(t *testing.T) {
		q9, ok := presetMap["quad9"]
		if !ok {
			t.Fatal("quad9 preset not found")
		}
		if q9.Primary != "9.9.9.9" {
			t.Errorf("Quad9 primary: expected 9.9.9.9, got %q", q9.Primary)
		}
		if q9.Secondary != "149.112.112.112" {
			t.Errorf("Quad9 secondary: expected 149.112.112.112, got %q", q9.Secondary)
		}
	})
}

func TestDNSPresetsNoDuplicateIDs(t *testing.T) {
	presets := GetDNSPresets()
	seen := make(map[string]bool)

	for _, p := range presets {
		if seen[p.ID] {
			t.Errorf("duplicate DNS preset ID: %q", p.ID)
		}
		seen[p.ID] = true
	}
}
