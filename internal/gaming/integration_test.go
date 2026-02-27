//go:build integration

package gaming

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"cleanforge/internal/gaming/profiles"
)

// TestProfileConsistency verifies every profile's tweak IDs exist in the global tweak catalog.
func TestProfileConsistency(t *testing.T) {
	// Build a set of all known tweak IDs from the catalog
	knownTweaks := make(map[string]bool, len(tweakCatalog))
	for _, td := range tweakCatalog {
		knownTweaks[td.ID] = true
	}

	allProfiles := profiles.AllProfiles()
	for _, profile := range allProfiles {
		for tweakID, enabled := range profile.Tweaks {
			if !enabled {
				continue
			}
			if !knownTweaks[tweakID] {
				t.Errorf("profile %q references unknown tweak ID %q", profile.ID, tweakID)
			}
		}
	}
}

// TestAllProfilesHaveRequiredTweaks verifies that each profile has at least 3 tweaks.
func TestAllProfilesHaveRequiredTweaks(t *testing.T) {
	allProfiles := profiles.AllProfiles()

	if len(allProfiles) == 0 {
		t.Fatal("AllProfiles() returned 0 profiles")
	}

	for _, profile := range allProfiles {
		enabledCount := 0
		for _, enabled := range profile.Tweaks {
			if enabled {
				enabledCount++
			}
		}
		if enabledCount < 3 {
			t.Errorf("profile %q has only %d enabled tweaks, want at least 3", profile.ID, enabledCount)
		}
	}
}

// TestNuclearHasEverything verifies that the Nuclear profile contains every tweak
// that appears in any other profile.
func TestNuclearHasEverything(t *testing.T) {
	allProfiles := profiles.AllProfiles()

	var nuclear *profiles.GameProfile
	for _, p := range allProfiles {
		if p.ID == "nuclear" {
			cp := p
			nuclear = &cp
			break
		}
	}

	if nuclear == nil {
		t.Fatal("nuclear profile not found")
	}

	// Collect all tweaks from all non-nuclear profiles
	allTweakIDs := make(map[string]string) // tweakID -> first profile that uses it
	for _, profile := range allProfiles {
		if profile.ID == "nuclear" {
			continue
		}
		for tweakID, enabled := range profile.Tweaks {
			if enabled {
				if _, exists := allTweakIDs[tweakID]; !exists {
					allTweakIDs[tweakID] = profile.ID
				}
			}
		}
	}

	// Verify nuclear has all of them
	for tweakID, fromProfile := range allTweakIDs {
		if !nuclear.Tweaks[tweakID] {
			t.Errorf("nuclear profile is missing tweak %q (present in %q)", tweakID, fromProfile)
		}
	}
}

// TestBackupRestoreFlow tests that the backup file mechanism works.
// It creates a GameBooster with a temp backup path and verifies backup
// file creation and the restore flow (reading the file back).
func TestBackupRestoreFlow(t *testing.T) {
	tmpDir := t.TempDir()
	backupPath := filepath.Join(tmpDir, "backup_state.json")

	gb := &GameBooster{
		appliedTweaks: make(map[string]bool),
		backupPath:    backupPath,
	}

	// Write a backup state manually (since we cannot access the real registry)
	state := &BackupState{
		CreatedAt: "2025-01-01T00:00:00Z",
		Entries: []BackupEntry{
			{
				Type:      "registry",
				Root:      "HKCU",
				KeyPath:   `Control Panel\Mouse`,
				ValueName: "MouseSpeed",
				Value:     "1",
				ValueType: 1, // REG_SZ
				Missing:   false,
			},
			{
				Type:         "service",
				ServiceName:  "SysMain",
				ServiceState: "running",
			},
		},
	}

	if err := gb.writeBackup(state); err != nil {
		t.Fatalf("writeBackup failed: %v", err)
	}

	// Verify backup file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Fatal("backup file was not created")
	}

	// Read back the backup
	loaded, err := gb.readBackup()
	if err != nil {
		t.Fatalf("readBackup failed: %v", err)
	}

	if loaded.CreatedAt != state.CreatedAt {
		t.Errorf("backup CreatedAt = %q, want %q", loaded.CreatedAt, state.CreatedAt)
	}
	if len(loaded.Entries) != 2 {
		t.Errorf("backup has %d entries, want 2", len(loaded.Entries))
	}

	// Verify first entry
	if len(loaded.Entries) > 0 {
		entry := loaded.Entries[0]
		if entry.Type != "registry" {
			t.Errorf("entry[0].Type = %q, want %q", entry.Type, "registry")
		}
		if entry.ValueName != "MouseSpeed" {
			t.Errorf("entry[0].ValueName = %q, want %q", entry.ValueName, "MouseSpeed")
		}
		if entry.Value != "1" {
			t.Errorf("entry[0].Value = %q, want %q", entry.Value, "1")
		}
	}

	// Verify second entry
	if len(loaded.Entries) > 1 {
		entry := loaded.Entries[1]
		if entry.Type != "service" {
			t.Errorf("entry[1].Type = %q, want %q", entry.Type, "service")
		}
		if entry.ServiceName != "SysMain" {
			t.Errorf("entry[1].ServiceName = %q, want %q", entry.ServiceName, "SysMain")
		}
		if entry.ServiceState != "running" {
			t.Errorf("entry[1].ServiceState = %q, want %q", entry.ServiceState, "running")
		}
	}
}

// TestBoostStatusTracking tests the boost status lifecycle: initially inactive,
// becomes active after profile is tracked, resets after restore.
func TestBoostStatusTracking(t *testing.T) {
	tmpDir := t.TempDir()
	backupPath := filepath.Join(tmpDir, "backup_state.json")

	gb := &GameBooster{
		appliedTweaks: make(map[string]bool),
		backupPath:    backupPath,
	}

	// Initially inactive
	status := gb.GetBoostStatus()
	if status.Active {
		t.Error("initial status.Active should be false")
	}
	if status.Profile != "" {
		t.Errorf("initial status.Profile = %q, want empty", status.Profile)
	}
	if len(status.TweaksApplied) != 0 {
		t.Errorf("initial status.TweaksApplied has %d items, want 0", len(status.TweaksApplied))
	}

	// Simulate applying a profile (set status directly since we cannot
	// actually apply registry tweaks in a test)
	gb.mu.Lock()
	gb.status = BoostStatus{
		Active:        true,
		Profile:       "competitive_fps",
		TweaksApplied: []string{"mouse_raw_input", "disable_game_dvr"},
		StartedAt:     "2025-01-01T00:00:00Z",
	}
	gb.appliedTweaks["mouse_raw_input"] = true
	gb.appliedTweaks["disable_game_dvr"] = true
	gb.mu.Unlock()

	// Verify active state
	status = gb.GetBoostStatus()
	if !status.Active {
		t.Error("status.Active should be true after profile apply")
	}
	if status.Profile != "competitive_fps" {
		t.Errorf("status.Profile = %q, want %q", status.Profile, "competitive_fps")
	}
	if len(status.TweaksApplied) != 2 {
		t.Errorf("status.TweaksApplied has %d items, want 2", len(status.TweaksApplied))
	}

	// Simulate restore (reset status)
	gb.mu.Lock()
	gb.status = BoostStatus{}
	gb.appliedTweaks = make(map[string]bool)
	gb.mu.Unlock()

	// Verify inactive after restore
	status = gb.GetBoostStatus()
	if status.Active {
		t.Error("status.Active should be false after restore")
	}
	if status.Profile != "" {
		t.Errorf("status.Profile = %q after restore, want empty", status.Profile)
	}
}

// TestGetProfilesReturnsAll verifies GetProfiles returns all profiles.
func TestGetProfilesReturnsAll(t *testing.T) {
	gb := &GameBooster{
		appliedTweaks: make(map[string]bool),
	}

	gps := gb.GetProfiles()
	expectedIDs := []string{
		"competitive_fps", "open_world", "moba_strategy",
		"racing_sim", "casual", "nuclear",
	}

	if len(gps) != len(expectedIDs) {
		t.Errorf("GetProfiles returned %d profiles, want %d", len(gps), len(expectedIDs))
	}

	idSet := make(map[string]bool)
	for _, gp := range gps {
		idSet[gp.ID] = true
	}

	for _, expectedID := range expectedIDs {
		if !idSet[expectedID] {
			t.Errorf("missing profile ID: %s", expectedID)
		}
	}
}

// TestGetAvailableTweaksReturnsAll verifies the available tweaks list
// matches the catalog length.
func TestGetAvailableTweaksReturnsAll(t *testing.T) {
	gb := &GameBooster{
		appliedTweaks: make(map[string]bool),
	}

	tweaks := gb.GetAvailableTweaks()
	if len(tweaks) != len(tweakCatalog) {
		t.Errorf("GetAvailableTweaks returned %d tweaks, want %d", len(tweaks), len(tweakCatalog))
	}

	for _, tw := range tweaks {
		if tw.ID == "" {
			t.Error("tweak has empty ID")
		}
		if tw.Name == "" {
			t.Error("tweak has empty Name")
		}
		if tw.Description == "" {
			t.Error("tweak has empty Description")
		}
		if tw.Category == "" {
			t.Error("tweak has empty Category")
		}
	}
}

// TestTweakCatalogHasUniqueIDs verifies there are no duplicate IDs in the tweak catalog.
func TestTweakCatalogHasUniqueIDs(t *testing.T) {
	seen := make(map[string]bool)
	for _, td := range tweakCatalog {
		if seen[td.ID] {
			t.Errorf("duplicate tweak ID: %s", td.ID)
		}
		seen[td.ID] = true
	}
}

// TestProfileFieldsPopulated verifies all profile fields are non-empty.
func TestProfileFieldsPopulated(t *testing.T) {
	allProfiles := profiles.AllProfiles()
	for _, p := range allProfiles {
		if p.ID == "" {
			t.Error("profile has empty ID")
		}
		if p.Name == "" {
			t.Errorf("profile %q has empty Name", p.ID)
		}
		if p.Description == "" {
			t.Errorf("profile %q has empty Description", p.ID)
		}
		if p.Icon == "" {
			t.Errorf("profile %q has empty Icon", p.ID)
		}
		if len(p.Tweaks) == 0 {
			t.Errorf("profile %q has no tweaks", p.ID)
		}
	}
}

// TestBackupStateJSON verifies the backup state round-trips through JSON correctly.
func TestBackupStateJSON(t *testing.T) {
	state := BackupState{
		CreatedAt: "2025-01-01T12:00:00Z",
		Entries: []BackupEntry{
			{
				Type:      "registry",
				Root:      "HKCU",
				KeyPath:   `Software\Test`,
				ValueName: "TestVal",
				Value:     "42",
				ValueType: 4, // REG_DWORD
			},
		},
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var loaded BackupState
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if loaded.CreatedAt != state.CreatedAt {
		t.Errorf("CreatedAt = %q, want %q", loaded.CreatedAt, state.CreatedAt)
	}
	if len(loaded.Entries) != 1 {
		t.Fatalf("Entries length = %d, want 1", len(loaded.Entries))
	}
	if loaded.Entries[0].ValueName != "TestVal" {
		t.Errorf("ValueName = %q, want %q", loaded.Entries[0].ValueName, "TestVal")
	}
}

// TestGetProfileByID verifies that GetProfileByID returns the correct profile
// or nil for unknown IDs.
func TestGetProfileByID(t *testing.T) {
	p := profiles.GetProfileByID("competitive_fps")
	if p == nil {
		t.Fatal("GetProfileByID('competitive_fps') returned nil")
	}
	if p.ID != "competitive_fps" {
		t.Errorf("profile ID = %q, want %q", p.ID, "competitive_fps")
	}

	// Unknown profile
	unknown := profiles.GetProfileByID("nonexistent_profile")
	if unknown != nil {
		t.Error("GetProfileByID('nonexistent_profile') should return nil")
	}
}

// TestIntegration_IsGUID verifies the isGUID helper function with additional cases.
func TestIntegration_IsGUID(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"e9a42b02-d5df-448d-aa00-03f14749eb61", true},
		{"381b4222-f694-41f0-9685-ff5bb260df2e", true},
		{"ABCDEF01-2345-6789-ABCD-EF0123456789", true},
		{"not-a-guid", false},
		{"", false},
		{"e9a42b02-d5df-448d-aa00", false},
		{"e9a42b02-d5df-448d-aa00-03f14749eb6g", false},
		{"12345678123412341234123456789012", false},
	}

	for _, tc := range tests {
		got := isGUID(tc.input)
		if got != tc.want {
			t.Errorf("isGUID(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}
