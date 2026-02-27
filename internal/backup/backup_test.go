package backup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetBackupPath(t *testing.T) {
	path := GetBackupPath()
	if path == "" {
		t.Fatal("GetBackupPath returned empty string")
	}

	if !strings.Contains(path, ".cleanforge") {
		t.Errorf("expected path to contain .cleanforge, got %q", path)
	}

	if !strings.Contains(path, "backups") {
		t.Errorf("expected path to contain backups, got %q", path)
	}

	// Directory should exist after calling GetBackupPath
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("backup directory should exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("backup path should be a directory")
	}
}

func TestNewEmptyState(t *testing.T) {
	s := newEmptyState()
	if s == nil {
		t.Fatal("newEmptyState returned nil")
	}

	if s.Timestamp == "" {
		t.Error("Timestamp is empty")
	}

	if s.RegistryKeys == nil {
		t.Error("RegistryKeys map is nil")
	}

	if s.Services == nil {
		t.Error("Services map is nil")
	}

	if len(s.RegistryKeys) != 0 {
		t.Error("RegistryKeys should be empty")
	}

	if len(s.Services) != 0 {
		t.Error("Services should be empty")
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Use a temp directory for the test
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, backupFilename)

	// Override the state with test data
	oldState := state
	defer func() { state = oldState }()

	state = &BackupState{
		Timestamp:    "2026-01-01T00:00:00Z",
		RegistryKeys: map[string]RegistryBackup{
			"HKCU\\Test\\Value": {
				Path:      "HKCU\\Test",
				ValueName: "Value",
				Value:     "test_data",
				Type:      "string",
				Existed:   true,
			},
		},
		Services:  map[string]string{"TestService": "AUTO_START"},
		PowerPlan: "381b4222-f694-41f0-9685-ff5bb260df2e",
	}

	// Save to temp file
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	// Verify file exists and is not empty
	info, err := os.Stat(tmpFile)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if info.Size() == 0 {
		t.Error("backup file is empty")
	}

	// Read it back
	readData, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	loaded := &BackupState{}
	if err := json.Unmarshal(readData, loaded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if loaded.Timestamp != "2026-01-01T00:00:00Z" {
		t.Errorf("expected timestamp %q, got %q", "2026-01-01T00:00:00Z", loaded.Timestamp)
	}

	if loaded.PowerPlan != "381b4222-f694-41f0-9685-ff5bb260df2e" {
		t.Errorf("expected powerPlan %q, got %q", "381b4222-f694-41f0-9685-ff5bb260df2e", loaded.PowerPlan)
	}

	if len(loaded.RegistryKeys) != 1 {
		t.Errorf("expected 1 registry key, got %d", len(loaded.RegistryKeys))
	}

	if loaded.Services["TestService"] != "AUTO_START" {
		t.Errorf("expected service TestService=AUTO_START, got %q", loaded.Services["TestService"])
	}
}

func TestParseRootKey(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"HKLM", false},
		{"HKCU", false},
		{"HKCR", false},
		{"HKU", false},
		{"HKCC", false},
		{"HKEY_LOCAL_MACHINE", false},
		{"HKEY_CURRENT_USER", false},
		{"INVALID", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := parseRootKey(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRootKey(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestSplitRegistryPath(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantRoot    string
		wantSubPath string
		wantErr     bool
	}{
		{
			name:        "Normal path",
			input:       `HKLM\SOFTWARE\Test`,
			wantRoot:    "HKLM",
			wantSubPath: `SOFTWARE\Test`,
		},
		{
			name:        "Deep path",
			input:       `HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\Run`,
			wantRoot:    "HKCU",
			wantSubPath: `SOFTWARE\Microsoft\Windows\CurrentVersion\Run`,
		},
		{
			name:    "No separator",
			input:   "HKLM",
			wantErr: true,
		},
		{
			name:        "Forward slashes",
			input:       "HKLM/SOFTWARE/Test",
			wantRoot:    "HKLM",
			wantSubPath: `SOFTWARE\Test`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, subPath, err := splitRegistryPath(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("splitRegistryPath(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if root != tt.wantRoot {
				t.Errorf("root = %q, want %q", root, tt.wantRoot)
			}
			// Normalize for comparison
			normalizedSubPath := strings.ReplaceAll(subPath, "/", "\\")
			normalizedWant := strings.ReplaceAll(tt.wantSubPath, "/", "\\")
			if normalizedSubPath != normalizedWant {
				t.Errorf("subPath = %q, want %q", subPath, tt.wantSubPath)
			}
		})
	}
}

func TestParseServiceStartType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Auto start",
			input:    "        START_TYPE         : 2   AUTO_START\n",
			expected: "AUTO_START",
		},
		{
			name:     "Demand start",
			input:    "        START_TYPE         : 3   DEMAND_START\n",
			expected: "DEMAND_START",
		},
		{
			name:     "Disabled",
			input:    "        START_TYPE         : 4   DISABLED\n",
			expected: "DISABLED",
		},
		{
			name:     "Full sc output",
			input:    "[SC] QueryServiceConfig SUCCESS\n\nSERVICE_NAME: SysMain\n        TYPE               : 20  WIN32_SHARE_PROCESS\n        START_TYPE         : 2   AUTO_START\n        ERROR_CONTROL      : 1   NORMAL\n",
			expected: "AUTO_START",
		},
		{
			name:     "No match",
			input:    "no start type here",
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
			result := parseServiceStartType(tt.input)
			if result != tt.expected {
				t.Errorf("parseServiceStartType = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestMapStartTypeToSC(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"AUTO_START", "auto"},
		{"DEMAND_START", "demand"},
		{"DISABLED", "disabled"},
		{"BOOT_START", "boot"},
		{"SYSTEM_START", "system"},
		{"auto", "auto"},
		{"demand", "demand"},
		{"AUTO", "auto"},
		{"DEMAND", "demand"},
		{"UNKNOWN", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapStartTypeToSC(tt.input)
			if result != tt.expected {
				t.Errorf("mapStartTypeToSC(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParsePowerPlanGUID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Standard output",
			input:    "Power Scheme GUID: 381b4222-f694-41f0-9685-ff5bb260df2e  (Balanced)",
			expected: "381b4222-f694-41f0-9685-ff5bb260df2e",
		},
		{
			name:     "No GUID keyword",
			input:    "No power scheme",
			expected: "",
		},
		{
			name:     "Empty",
			input:    "",
			expected: "",
		},
		{
			name:     "Ultimate Performance",
			input:    "Power Scheme GUID: e9a42b02-d5df-448d-aa00-03f14749eb61  (Ultimate Performance)",
			expected: "e9a42b02-d5df-448d-aa00-03f14749eb61",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parsePowerPlanGUID(tt.input)
			if result != tt.expected {
				t.Errorf("parsePowerPlanGUID = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestHasBackup(t *testing.T) {
	// Simply verify it does not panic
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("HasBackup panicked: %v", r)
		}
	}()

	result := HasBackup()
	t.Logf("HasBackup: %v", result)
}

func TestRestoreRegistryEmptyState(t *testing.T) {
	// Restore with empty state should be a no-op
	oldState := state
	defer func() { state = oldState }()

	state = newEmptyState()
	err := RestoreRegistry()
	if err != nil {
		t.Errorf("RestoreRegistry with empty state should not error: %v", err)
	}
}

func TestRestoreServicesEmptyState(t *testing.T) {
	oldState := state
	defer func() { state = oldState }()

	state = newEmptyState()
	err := RestoreServices()
	if err != nil {
		t.Errorf("RestoreServices with empty state should not error: %v", err)
	}
}

func TestRestorePowerPlanEmptyState(t *testing.T) {
	oldState := state
	defer func() { state = oldState }()

	state = newEmptyState()
	err := RestorePowerPlan()
	if err != nil {
		t.Errorf("RestorePowerPlan with empty state should not error: %v", err)
	}
}

func TestRestoreAllEmptyState(t *testing.T) {
	oldState := state
	defer func() { state = oldState }()

	state = newEmptyState()
	err := RestoreAll()
	if err != nil {
		t.Errorf("RestoreAll with empty state should not error: %v", err)
	}
}

func TestBackupStateJSON(t *testing.T) {
	s := &BackupState{
		Timestamp: "2026-01-01T00:00:00Z",
		RegistryKeys: map[string]RegistryBackup{
			"test": {
				Path:      "HKCU\\Test",
				ValueName: "TestVal",
				Value:     uint32(42),
				Type:      "dword",
				Existed:   true,
			},
		},
		Services:  map[string]string{"svc1": "AUTO_START"},
		PowerPlan: "abc-123",
	}

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var loaded BackupState
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if loaded.Timestamp != s.Timestamp {
		t.Errorf("timestamp mismatch: %q != %q", loaded.Timestamp, s.Timestamp)
	}
	if loaded.PowerPlan != s.PowerPlan {
		t.Errorf("powerPlan mismatch: %q != %q", loaded.PowerPlan, s.PowerPlan)
	}
}
