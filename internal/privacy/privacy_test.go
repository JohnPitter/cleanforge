package privacy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAllTweaksNotEmpty(t *testing.T) {
	if len(allTweaks) == 0 {
		t.Fatal("allTweaks should not be empty")
	}
	if len(allTweaks) < 12 {
		t.Errorf("expected at least 12 privacy tweaks, got %d", len(allTweaks))
	}
}

func TestAllTweaksHaveRequiredFields(t *testing.T) {
	seenIDs := make(map[string]bool)
	validCategories := map[string]bool{
		"telemetry": true,
		"tracking":  true,
		"ads":       true,
		"cortana":   true,
	}

	for _, tw := range allTweaks {
		t.Run(tw.id, func(t *testing.T) {
			if tw.id == "" {
				t.Error("tweak has empty id")
			}
			if tw.name == "" {
				t.Errorf("tweak %q has empty name", tw.id)
			}
			if tw.description == "" {
				t.Errorf("tweak %q has empty description", tw.id)
			}
			if !validCategories[tw.category] {
				t.Errorf("tweak %q has invalid category %q", tw.id, tw.category)
			}
			if seenIDs[tw.id] {
				t.Errorf("duplicate tweak ID: %q", tw.id)
			}
			seenIDs[tw.id] = true
		})
	}
}

func TestTelemetryHostsNotEmpty(t *testing.T) {
	if len(telemetryHosts) == 0 {
		t.Fatal("telemetryHosts should not be empty")
	}

	for _, host := range telemetryHosts {
		if host == "" {
			t.Error("telemetryHosts contains empty string")
		}
		if !strings.Contains(host, ".") {
			t.Errorf("telemetryHost %q doesn't look like a domain", host)
		}
	}
}

func TestGetPrivacyTweaks(t *testing.T) {
	tweaks, err := GetPrivacyTweaks()
	if err != nil {
		t.Fatalf("GetPrivacyTweaks returned error: %v", err)
	}

	if len(tweaks) != len(allTweaks) {
		t.Errorf("expected %d tweaks, got %d", len(allTweaks), len(tweaks))
	}

	for _, tw := range tweaks {
		if tw.ID == "" {
			t.Error("PrivacyTweak has empty ID")
		}
		if tw.Name == "" {
			t.Errorf("PrivacyTweak %q has empty Name", tw.ID)
		}
		if tw.Category == "" {
			t.Errorf("PrivacyTweak %q has empty Category", tw.ID)
		}
	}
}

func TestGetHostsFilePath(t *testing.T) {
	path := getHostsFilePath()
	if path == "" {
		t.Fatal("getHostsFilePath returned empty string")
	}

	if !strings.HasSuffix(path, "hosts") {
		t.Errorf("expected path to end with 'hosts', got %q", path)
	}

	if !strings.Contains(path, "drivers") || !strings.Contains(path, "etc") {
		t.Errorf("expected path to contain 'drivers/etc', got %q", path)
	}
}

func TestHostsFileMarkers(t *testing.T) {
	if hostsFileMarkerStart == "" {
		t.Error("hostsFileMarkerStart is empty")
	}
	if hostsFileMarkerEnd == "" {
		t.Error("hostsFileMarkerEnd is empty")
	}
	if hostsFileMarkerStart == hostsFileMarkerEnd {
		t.Error("start and end markers should be different")
	}
	if !strings.HasPrefix(hostsFileMarkerStart, "#") {
		t.Error("start marker should be a comment (start with #)")
	}
	if !strings.HasPrefix(hostsFileMarkerEnd, "#") {
		t.Error("end marker should be a comment (start with #)")
	}
}

func TestApplyTweakUnknownID(t *testing.T) {
	err := ApplyTweak("nonexistent_tweak_id")
	if err == nil {
		t.Error("ApplyTweak with unknown ID should return error")
	}
}

func TestHostsBlockAppliedOnCleanSystem(t *testing.T) {
	// On a clean system, hosts block should not be applied
	// This test checks if it does not panic
	result := isHostsBlockApplied()
	t.Logf("isHostsBlockApplied: %v", result)
}

func TestHostsBlockLifecycle(t *testing.T) {
	// Create a temp hosts file for testing
	tmpDir := t.TempDir()
	tmpHosts := filepath.Join(tmpDir, "hosts")

	// Write initial content
	initialContent := "# Hosts file\n127.0.0.1 localhost\n"
	if err := os.WriteFile(tmpHosts, []byte(initialContent), 0644); err != nil {
		t.Fatalf("failed to write temp hosts: %v", err)
	}

	// We can't easily test the real hosts file functions without modifying
	// getHostsFilePath, but we can verify the marker logic
	t.Run("MarkersDetection", func(t *testing.T) {
		// Write content with markers
		content := initialContent + "\n" + hostsFileMarkerStart + "\n0.0.0.0 test.example.com\n" + hostsFileMarkerEnd + "\n"
		if err := os.WriteFile(tmpHosts, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write: %v", err)
		}

		data, _ := os.ReadFile(tmpHosts)
		if !strings.Contains(string(data), hostsFileMarkerStart) {
			t.Error("hosts file should contain start marker")
		}
		if !strings.Contains(string(data), hostsFileMarkerEnd) {
			t.Error("hosts file should contain end marker")
		}
	})
}

func TestPrivacyTweakIDs(t *testing.T) {
	tweaks, err := GetPrivacyTweaks()
	if err != nil {
		t.Fatalf("GetPrivacyTweaks returned error: %v", err)
	}

	expectedIDs := map[string]bool{
		"disable_telemetry":            true,
		"disable_activity_history":     true,
		"disable_location":             true,
		"disable_advertising_id":       true,
		"disable_cortana":              true,
		"disable_bing_search":          true,
		"disable_feedback":             true,
		"disable_tailored_experiences": true,
		"disable_tips":                 true,
		"block_telemetry_hosts":        true,
		"disable_wifi_sense":           true,
		"disable_error_reporting":      true,
	}

	foundIDs := make(map[string]bool)
	for _, tw := range tweaks {
		foundIDs[tw.ID] = true
	}

	for id := range expectedIDs {
		if !foundIDs[id] {
			t.Errorf("expected privacy tweak ID %q not found", id)
		}
	}

	for id := range foundIDs {
		if !expectedIDs[id] {
			t.Errorf("unexpected privacy tweak ID %q", id)
		}
	}
}

func TestPrivacyTweakCategories(t *testing.T) {
	tweaks, err := GetPrivacyTweaks()
	if err != nil {
		t.Fatalf("GetPrivacyTweaks returned error: %v", err)
	}

	validCategories := map[string]bool{
		"telemetry": true,
		"tracking":  true,
		"ads":       true,
		"cortana":   true,
	}

	categoryCounts := make(map[string]int)
	for _, tw := range tweaks {
		if !validCategories[tw.Category] {
			t.Errorf("tweak %q has unexpected category %q", tw.ID, tw.Category)
		}
		categoryCounts[tw.Category]++
	}

	// Verify all expected categories are present
	for cat := range validCategories {
		if categoryCounts[cat] == 0 {
			t.Errorf("no tweaks found in category %q", cat)
		}
	}

	t.Logf("Category distribution: %v", categoryCounts)
}

func TestPrivacyTweakFields(t *testing.T) {
	tweaks, err := GetPrivacyTweaks()
	if err != nil {
		t.Fatalf("GetPrivacyTweaks returned error: %v", err)
	}

	for _, tw := range tweaks {
		t.Run("Fields_"+tw.ID, func(t *testing.T) {
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
		})
	}
}

func TestPrivacyTweakCount(t *testing.T) {
	tweaks, err := GetPrivacyTweaks()
	if err != nil {
		t.Fatalf("GetPrivacyTweaks returned error: %v", err)
	}

	if len(tweaks) != 12 {
		t.Errorf("expected exactly 12 privacy tweaks, got %d", len(tweaks))
	}
}

func TestLocationTweakDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("isLocationTweakApplied panicked: %v", r)
		}
	}()

	_ = isLocationTweakApplied()
}
