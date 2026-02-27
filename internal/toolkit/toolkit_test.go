package toolkit

import (
	"strings"
	"testing"
)

func TestIsAdmin(t *testing.T) {
	// This test simply verifies IsAdmin does not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("IsAdmin panicked: %v", r)
		}
	}()

	result := IsAdmin()
	t.Logf("IsAdmin: %v", result)
}

func TestKnownBloatwareNotEmpty(t *testing.T) {
	if len(knownBloatware) == 0 {
		t.Error("knownBloatware list is empty")
	}
	if len(knownBloatware) < 20 {
		t.Errorf("expected at least 20 bloatware entries, got %d", len(knownBloatware))
	}
}

func TestFriendlyName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Microsoft.BingWeather", "Bing Weather"},
		{"Microsoft.YourPhone", "Your Phone / Phone Link"},
		{"Microsoft.549981C3F5F10", "Cortana"},
		{"SpotifyAB.SpotifyMusic", "Spotify"},
		{"king.com.CandyCrushSaga", "Candy Crush Saga"},
		{"BytedancePte.Ltd.TikTok", "TikTok"},
		{"Unknown.Package", "Package"},
		{"SingleWord", "SingleWord"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := friendlyName(tt.input)
			if result != tt.expected {
				t.Errorf("friendlyName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFriendlyNameCoversAllKnownBloatware(t *testing.T) {
	for _, pkg := range knownBloatware {
		name := friendlyName(pkg)
		if name == "" {
			t.Errorf("friendlyName(%q) returned empty string", pkg)
		}
		if name == pkg {
			t.Errorf("friendlyName(%q) returned the raw package name; should have a friendly name", pkg)
		}
	}
}

func TestParseCSVLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Simple fields",
			input:    `a,b,c`,
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "Quoted fields",
			input:    `"Name","PackageFullName","Publisher"`,
			expected: []string{"Name", "PackageFullName", "Publisher"},
		},
		{
			name:     "Mixed quoted and unquoted",
			input:    `"test",value,"another"`,
			expected: []string{"test", "value", "another"},
		},
		{
			name:     "Empty fields",
			input:    `,,`,
			expected: []string{"", "", ""},
		},
		{
			name:     "Single field",
			input:    `hello`,
			expected: []string{"hello"},
		},
		{
			name:     "Comma in quotes",
			input:    `"hello, world",test`,
			expected: []string{"hello, world", "test"},
		},
		{
			name:     "Escaped quote in field",
			input:    `"he said ""hi""",test`,
			expected: []string{`he said "hi"`, "test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCSVLine(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("parseCSVLine(%q) returned %d fields, want %d: %v", tt.input, len(result), len(tt.expected), result)
				return
			}
			for i, field := range result {
				if field != tt.expected[i] {
					t.Errorf("parseCSVLine(%q)[%d] = %q, want %q", tt.input, i, field, tt.expected[i])
				}
			}
		})
	}
}

func TestRemoveBloatwareEmptyList(t *testing.T) {
	result, err := RemoveBloatware([]string{})
	if err != nil {
		t.Fatalf("RemoveBloatware with empty list returned error: %v", err)
	}
	if result == nil {
		t.Fatal("RemoveBloatware returned nil result")
	}
	if !result.Success {
		t.Error("RemoveBloatware with empty list should succeed")
	}
}

func TestRunSFCRequiresAdmin(t *testing.T) {
	// When not running as admin, SFC should return a result indicating admin required
	if IsAdmin() {
		t.Skip("skipping: running as admin")
	}

	result, err := RunSFC()
	if err != nil {
		t.Fatalf("RunSFC returned error: %v", err)
	}
	if result == nil {
		t.Fatal("RunSFC returned nil result")
	}
	if result.Success {
		t.Error("RunSFC should not succeed without admin privileges")
	}
	if !strings.Contains(result.Output, "administrator") {
		t.Errorf("expected admin warning in output, got: %q", result.Output)
	}
}

func TestRunDISMRequiresAdmin(t *testing.T) {
	if IsAdmin() {
		t.Skip("skipping: running as admin")
	}

	result, err := RunDISM()
	if err != nil {
		t.Fatalf("RunDISM returned error: %v", err)
	}
	if result == nil {
		t.Fatal("RunDISM returned nil result")
	}
	if result.Success {
		t.Error("RunDISM should not succeed without admin privileges")
	}
}

func TestRebuildFontCacheRequiresAdmin(t *testing.T) {
	if IsAdmin() {
		t.Skip("skipping: running as admin")
	}

	result, err := RebuildFontCache()
	if err != nil {
		t.Fatalf("RebuildFontCache returned error: %v", err)
	}
	if result.Success {
		t.Error("RebuildFontCache should not succeed without admin")
	}
}

func TestResetWindowsSearchRequiresAdmin(t *testing.T) {
	if IsAdmin() {
		t.Skip("skipping: running as admin")
	}

	result, err := ResetWindowsSearch()
	if err != nil {
		t.Fatalf("ResetWindowsSearch returned error: %v", err)
	}
	if result.Success {
		t.Error("ResetWindowsSearch should not succeed without admin")
	}
}

func TestRepairWindowsUpdateRequiresAdmin(t *testing.T) {
	if IsAdmin() {
		t.Skip("skipping: running as admin")
	}

	result, err := RepairWindowsUpdate()
	if err != nil {
		t.Fatalf("RepairWindowsUpdate returned error: %v", err)
	}
	if result.Success {
		t.Error("RepairWindowsUpdate should not succeed without admin")
	}
}

func TestGetBloatwareApps(t *testing.T) {
	apps, err := GetBloatwareApps()
	if err != nil {
		// PowerShell query may fail in test environments
		t.Logf("GetBloatwareApps returned error (may be expected in test env): %v", err)
		return
	}

	if apps == nil {
		t.Fatal("GetBloatwareApps returned nil without error")
	}

	// Verify the returned list has entries matching the knownBloatware list
	if len(apps) != len(knownBloatware) {
		t.Errorf("expected %d bloatware apps, got %d", len(knownBloatware), len(apps))
	}

	for _, app := range apps {
		if app.Name == "" {
			t.Errorf("bloatware app has empty Name for package %q", app.PackageName)
		}
		if app.PackageName == "" {
			t.Error("bloatware app has empty PackageName")
		}
	}
}

func TestGetBloatwareAppsContainsKnownNames(t *testing.T) {
	apps, err := GetBloatwareApps()
	if err != nil {
		t.Skipf("GetBloatwareApps returned error: %v", err)
	}

	// Build a map of returned package names
	appMap := make(map[string]BloatwareApp)
	for _, app := range apps {
		appMap[app.PackageName] = app
	}

	// Verify known bloatware entries appear with friendly names
	knownEntries := map[string]string{
		"Microsoft.BingWeather":                  "Bing Weather",
		"Microsoft.MicrosoftSolitaireCollection": "Solitaire Collection",
		"Microsoft.549981C3F5F10":                "Cortana",
		"SpotifyAB.SpotifyMusic":                 "Spotify",
	}

	for pkgName, expectedName := range knownEntries {
		app, ok := appMap[pkgName]
		if !ok {
			t.Errorf("expected package %q in bloatware apps list", pkgName)
			continue
		}
		if app.Name != expectedName {
			t.Errorf("package %q: expected name %q, got %q", pkgName, expectedName, app.Name)
		}
	}
}

func TestFriendlyNameFallback(t *testing.T) {
	// Test the fallback behavior for unknown package names
	tests := []struct {
		input    string
		expected string
	}{
		{"Unknown.SomeApp", "SomeApp"},
		{"Publisher.AppName", "AppName"},
		{"SingleWord", "SingleWord"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := friendlyName(tt.input)
			if result != tt.expected {
				t.Errorf("friendlyName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToolResultStruct(t *testing.T) {
	result := &ToolResult{
		Name:    "Test Tool",
		Success: true,
		Output:  "operation completed",
		Errors:  nil,
	}

	if result.Name != "Test Tool" {
		t.Errorf("expected name %q, got %q", "Test Tool", result.Name)
	}
	if !result.Success {
		t.Error("expected success=true")
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected no errors, got %v", result.Errors)
	}
}
