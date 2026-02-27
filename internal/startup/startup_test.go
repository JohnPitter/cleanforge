package startup

import (
	"testing"
)

func TestNewStartupManager(t *testing.T) {
	sm := NewStartupManager()
	if sm == nil {
		t.Fatal("NewStartupManager returned nil")
	}
}

func TestGetStartupItems(t *testing.T) {
	sm := NewStartupManager()

	items, err := sm.GetStartupItems()
	if err != nil {
		t.Fatalf("GetStartupItems returned error: %v", err)
	}

	// On a real Windows system there should be some startup items,
	// but on CI there may be none. We just verify it returns a valid slice.
	if items == nil {
		t.Error("GetStartupItems returned nil; expected an empty or populated slice")
	}

	t.Logf("Found %d startup items", len(items))

	// If there are items, validate their fields
	for i, item := range items {
		if item.Name == "" {
			t.Errorf("item[%d] has empty Name", i)
		}
		if item.Location == "" {
			t.Errorf("item[%d] (%s) has empty Location", i, item.Name)
		}
		validLocations := map[string]bool{
			"registry_hkcu":  true,
			"registry_hklm":  true,
			"startup_folder": true,
			"task_scheduler": true,
		}
		if !validLocations[item.Location] {
			t.Errorf("item[%d] (%s) has unexpected location: %q", i, item.Name, item.Location)
		}
	}
}

func TestEstimateImpact(t *testing.T) {
	sm := NewStartupManager()

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "Teams executable - high impact",
			path:     `C:\Program Files\Microsoft Teams\Teams.exe`,
			expected: "high",
		},
		{
			name:     "Spotify executable - medium impact",
			path:     `C:\Program Files\Spotify\Spotify.exe`,
			expected: "medium",
		},
		{
			name:     "Discord executable - medium impact",
			path:     `C:\Users\test\AppData\Local\Discord\discord.exe`,
			expected: "medium",
		},
		{
			name:     "Steam executable - medium impact",
			path:     `C:\Program Files (x86)\Steam\steam.exe`,
			expected: "medium",
		},
		{
			name:     "Chrome executable - high impact",
			path:     `"C:\Program Files\Google\Chrome\Application\chrome.exe" --no-startup-window`,
			expected: "high",
		},
		{
			name:     "Empty path returns unknown",
			path:     "",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sm.EstimateImpact(tt.path)
			if result != tt.expected {
				t.Errorf("EstimateImpact(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}

	t.Run("Unknown path returns something valid", func(t *testing.T) {
		result := sm.EstimateImpact(`C:\SomeRandom\Unknown\app.exe`)
		validImpacts := map[string]bool{
			"high":    true,
			"medium":  true,
			"low":     true,
			"unknown": true,
		}
		if !validImpacts[result] {
			t.Errorf("EstimateImpact returned unexpected value: %q", result)
		}
	})
}

func TestExtractExePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Quoted path with args",
			input:    `"C:\Program Files\App\app.exe" --flag1 --flag2`,
			expected: `C:\Program Files\App\app.exe`,
		},
		{
			name:     "Quoted path without args",
			input:    `"C:\Program Files\App\app.exe"`,
			expected: `C:\Program Files\App\app.exe`,
		},
		{
			name:     "Unquoted path with exe",
			input:    `C:\App\myapp.exe -silent`,
			expected: `C:\App\myapp.exe`,
		},
		{
			name:     "Unquoted simple path",
			input:    `C:\App\myapp.exe`,
			expected: `C:\App\myapp.exe`,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Path without exe extension",
			input:    `C:\App\myapp --flag`,
			expected: `C:\App\myapp`,
		},
		{
			name:     "Spaces in path only",
			input:    `   `,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractExePath(tt.input)
			if result != tt.expected {
				t.Errorf("extractExePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractPublisher(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "Microsoft path",
			path:     `C:\Program Files\Microsoft Teams\Teams.exe`,
			expected: "Microsoft Corporation",
		},
		{
			name:     "Google path",
			path:     `C:\Program Files\Google\Chrome\Application\chrome.exe`,
			expected: "Google LLC",
		},
		{
			name:     "NVIDIA path",
			path:     `C:\Program Files\NVIDIA Corporation\Display\nvtray.exe`,
			expected: "NVIDIA Corporation",
		},
		{
			name:     "Steam path",
			path:     `C:\Program Files (x86)\Steam\Steam.exe`,
			expected: "Valve Corporation",
		},
		{
			name:     "Discord path",
			path:     `C:\Users\test\AppData\Local\Discord\Update.exe`,
			expected: "Discord Inc.",
		},
		{
			name:     "Spotify path",
			path:     `C:\Users\test\AppData\Roaming\Spotify\Spotify.exe`,
			expected: "Spotify AB",
		},
		{
			name:     "OneDrive path",
			path:     `C:\Users\test\AppData\Local\Microsoft\OneDrive\OneDrive.exe`,
			expected: "Microsoft Corporation",
		},
		{
			name:     "Unknown publisher",
			path:     `C:\CustomApp\SomeUnknown\randomtool.exe`,
			expected: "",
		},
		{
			name:     "Quoted path with Microsoft",
			path:     `"C:\Program Files\Microsoft Office\Office16\WINWORD.EXE"`,
			expected: "Microsoft Corporation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPublisher(tt.path)
			if result != tt.expected {
				t.Errorf("extractPublisher(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
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
			input:    `"field1","field2","field3"`,
			expected: []string{"field1", "field2", "field3"},
		},
		{
			name:     "Fields with spaces",
			input:    `"field one","field two","field three"`,
			expected: []string{"field one", "field two", "field three"},
		},
		{
			name:     "Empty fields",
			input:    `"","",""`,
			expected: []string{"", "", ""},
		},
		{
			name:     "Single field",
			input:    `"single"`,
			expected: []string{"single"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCSVLine(tt.input)
			if len(result) != len(tt.expected) {
				t.Fatalf("parseCSVLine(%q): got %d fields, want %d: %v",
					tt.input, len(result), len(tt.expected), result)
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("parseCSVLine(%q)[%d] = %q, want %q", tt.input, i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestKnownImpactMap(t *testing.T) {
	// Verify that known high-impact executables are present
	highImpact := []string{"teams.exe", "onedrive.exe", "msedge.exe", "chrome.exe", "slack.exe"}
	for _, exe := range highImpact {
		impact, ok := knownImpact[exe]
		if !ok {
			t.Errorf("expected %q in knownImpact map", exe)
			continue
		}
		if impact != "high" {
			t.Errorf("expected %q to be 'high', got %q", exe, impact)
		}
	}

	// Verify medium-impact entries
	mediumImpact := []string{"spotify.exe", "discord.exe", "steam.exe"}
	for _, exe := range mediumImpact {
		impact, ok := knownImpact[exe]
		if !ok {
			t.Errorf("expected %q in knownImpact map", exe)
			continue
		}
		if impact != "medium" {
			t.Errorf("expected %q to be 'medium', got %q", exe, impact)
		}
	}
}

func TestIsStartupTask(t *testing.T) {
	tests := []struct {
		name     string
		taskName string
		expected bool
	}{
		{"Google update task", `\GoogleUpdateTask`, true},
		{"Adobe updater", `\AdobeAcrobatUpdateTask`, true},
		{"Microsoft Teams startup", `\Microsoft\Office\TeamsStartup`, true},
		{"Random task", `\MyCustomScheduledJob`, false},
		{"Startup keyword", `\AutoStartHelper`, true},
		{"Discord helper", `\DiscordHelper`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isStartupTask(tt.taskName)
			if result != tt.expected {
				t.Errorf("isStartupTask(%q) = %v, want %v", tt.taskName, result, tt.expected)
			}
		})
	}
}
