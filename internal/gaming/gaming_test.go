package gaming

import (
	"testing"
)

func TestNewGameBooster(t *testing.T) {
	gb := NewGameBooster()

	if gb == nil {
		t.Fatal("NewGameBooster returned nil")
	}

	if gb.appliedTweaks == nil {
		t.Error("appliedTweaks map not initialized")
	}

	if gb.backupPath == "" {
		t.Error("backupPath is empty")
	}
}

func TestGetBoostStatus(t *testing.T) {
	gb := NewGameBooster()

	status := gb.GetBoostStatus()
	if status == nil {
		t.Fatal("GetBoostStatus returned nil")
	}

	if status.Active {
		t.Error("default boost status should be inactive")
	}

	if status.Profile != "" {
		t.Errorf("default profile should be empty, got %q", status.Profile)
	}

	if len(status.TweaksApplied) != 0 {
		t.Errorf("default tweaks applied should be empty, got %v", status.TweaksApplied)
	}

	if status.StartedAt != "" {
		t.Errorf("default started at should be empty, got %q", status.StartedAt)
	}
}

func TestDetectGPU(t *testing.T) {
	// This test verifies DetectGPU does not panic.
	// It may fail on CI or in VMs, but it should not crash.
	gb := NewGameBooster()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("DetectGPU panicked: %v", r)
		}
	}()

	gpuInfo, err := gb.DetectGPU()
	if err != nil {
		// Not a failure - may not have a detectable GPU in test env
		t.Logf("DetectGPU returned error (may be expected in test env): %v", err)
		return
	}

	if gpuInfo == nil {
		t.Error("DetectGPU returned nil info without error")
		return
	}

	t.Logf("Detected GPU: Name=%q, Vendor=%q, Driver=%q, ProfileName=%q",
		gpuInfo.Name, gpuInfo.Vendor, gpuInfo.Driver, gpuInfo.ProfileName)

	// If we got a result, validate the vendor is one of the known values
	validVendors := map[string]bool{
		"nvidia":  true,
		"amd":     true,
		"intel":   true,
		"unknown": true,
	}
	if !validVendors[gpuInfo.Vendor] {
		t.Errorf("unexpected GPU vendor: %q", gpuInfo.Vendor)
	}
}

func TestKillBloatware(t *testing.T) {
	// This test verifies KillBloatware returns a list without panicking.
	// On a test system, it may not kill anything.
	gb := NewGameBooster()

	t.Run("NonAggressive", func(t *testing.T) {
		killed, err := gb.KillBloatware(false)
		if err != nil {
			t.Fatalf("KillBloatware(false) returned error: %v", err)
		}
		// killed may be empty; that is fine
		t.Logf("Non-aggressive: killed %d processes", len(killed))
	})

	t.Run("Aggressive", func(t *testing.T) {
		killed, err := gb.KillBloatware(true)
		if err != nil {
			t.Fatalf("KillBloatware(true) returned error: %v", err)
		}
		t.Logf("Aggressive: killed %d processes", len(killed))
	})
}

func TestGetAvailableTweaks(t *testing.T) {
	gb := NewGameBooster()

	tweaks := gb.GetAvailableTweaks()
	if len(tweaks) < 24 {
		t.Errorf("expected at least 24 tweaks, got %d", len(tweaks))
	}

	// Verify each tweak has required fields
	seenIDs := make(map[string]bool)
	for _, tw := range tweaks {
		if tw.ID == "" {
			t.Error("tweak has empty ID")
		}
		if tw.Name == "" {
			t.Errorf("tweak %q has empty Name", tw.ID)
		}
		if tw.Description == "" {
			t.Errorf("tweak %q has empty Description", tw.ID)
		}
		if tw.Category == "" {
			t.Errorf("tweak %q has empty Category", tw.ID)
		}
		if seenIDs[tw.ID] {
			t.Errorf("duplicate tweak ID: %q", tw.ID)
		}
		seenIDs[tw.ID] = true
	}

	// Verify known tweak categories
	validCategories := map[string]bool{
		"mouse":    true,
		"keyboard": true,
		"gpu":      true,
		"display":  true,
		"power":    true,
		"system":   true,
		"network":  true,
	}
	for _, tw := range tweaks {
		if !validCategories[tw.Category] {
			t.Errorf("tweak %q has unexpected category %q", tw.ID, tw.Category)
		}
	}
}

func TestGetProfiles(t *testing.T) {
	gb := NewGameBooster()

	profiles := gb.GetProfiles()
	if len(profiles) != 6 {
		t.Errorf("expected 6 profiles, got %d", len(profiles))
	}

	expectedIDs := map[string]bool{
		"competitive_fps": true,
		"open_world":      true,
		"moba_strategy":   true,
		"racing_sim":      true,
		"casual":          true,
		"nuclear":         true,
	}

	for _, p := range profiles {
		if !expectedIDs[p.ID] {
			t.Errorf("unexpected profile ID: %q", p.ID)
		}
		if p.Name == "" {
			t.Errorf("profile %q has empty Name", p.ID)
		}
		if p.Icon == "" {
			t.Errorf("profile %q has empty Icon", p.ID)
		}
		if p.Description == "" {
			t.Errorf("profile %q has empty Description", p.ID)
		}
		if len(p.Tweaks) == 0 {
			t.Errorf("profile %q has no tweaks", p.ID)
		}
	}
}

func TestIsGUID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"Valid GUID", "e9a42b02-d5df-448d-aa00-03f14749eb61", true},
		{"Valid GUID uppercase", "E9A42B02-D5DF-448D-AA00-03F14749EB61", true},
		{"Too short", "e9a42b02-d5df-448d-aa00", false},
		{"No dashes", "e9a42b02d5df448daa0003f14749eb61", false},
		{"Empty", "", false},
		{"Invalid chars", "e9a42b02-d5df-448d-aa00-03f14749eg61", false},
		{"Wrong dash positions", "e9a42b0-2d5df-448d-aa00-03f14749eb61", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isGUID(tt.input)
			if result != tt.expected {
				t.Errorf("isGUID(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseGUIDFromPowercfg(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Standard output",
			input:    "Power Scheme GUID: e9a42b02-d5df-448d-aa00-03f14749eb61 (Ultimate Performance)",
			expected: "e9a42b02-d5df-448d-aa00-03f14749eb61",
		},
		{
			name:     "No GUID",
			input:    "No power scheme found",
			expected: "",
		},
		{
			name:     "Empty",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseGUIDFromPowercfg(tt.input)
			if result != tt.expected {
				t.Errorf("parseGUIDFromPowercfg(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFindUltimatePlanGUID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "Standard list with ultimate plan",
			input: `Existing Power Schemes (* denotes currently active)
Power Scheme GUID: 381b4222-f694-41f0-9685-ff5bb260df2e  (Balanced) *
Power Scheme GUID: e9a42b02-d5df-448d-aa00-03f14749eb61  (Ultimate Performance)`,
			expected: "e9a42b02-d5df-448d-aa00-03f14749eb61",
		},
		{
			name:     "No ultimate plan",
			input:    `Power Scheme GUID: 381b4222-f694-41f0-9685-ff5bb260df2e  (Balanced) *`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findUltimatePlanGUID(tt.input)
			if result != tt.expected {
				t.Errorf("findUltimatePlanGUID = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestBloatwareLists(t *testing.T) {
	t.Run("HeavyBloatwareNotEmpty", func(t *testing.T) {
		if len(heavyBloatware) == 0 {
			t.Error("heavyBloatware list is empty")
		}
	})

	t.Run("LightBloatwareNotEmpty", func(t *testing.T) {
		if len(lightBloatware) == 0 {
			t.Error("lightBloatware list is empty")
		}
	})

	t.Run("LightIsSubsetOfHeavy", func(t *testing.T) {
		heavySet := make(map[string]bool)
		for _, name := range heavyBloatware {
			heavySet[name] = true
		}

		for _, name := range lightBloatware {
			if !heavySet[name] {
				t.Errorf("light bloatware %q is not in heavy bloatware list", name)
			}
		}
	})

	t.Run("HeavyHasMoreEntries", func(t *testing.T) {
		if len(heavyBloatware) <= len(lightBloatware) {
			t.Errorf("heavy bloatware (%d) should have more entries than light (%d)",
				len(heavyBloatware), len(lightBloatware))
		}
	})
}

func TestTweakCatalog(t *testing.T) {
	if len(tweakCatalog) < 24 {
		t.Errorf("expected at least 24 tweaks in catalog, got %d", len(tweakCatalog))
	}

	seenIDs := make(map[string]bool)
	for _, tw := range tweakCatalog {
		if tw.ID == "" {
			t.Error("tweak in catalog has empty ID")
		}
		if tw.Name == "" {
			t.Errorf("tweak %q has empty Name", tw.ID)
		}
		if tw.Category == "" {
			t.Errorf("tweak %q has empty Category", tw.ID)
		}
		if seenIDs[tw.ID] {
			t.Errorf("duplicate tweak ID in catalog: %q", tw.ID)
		}
		seenIDs[tw.ID] = true
	}
}
