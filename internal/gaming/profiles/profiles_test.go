package profiles

import (
	"testing"
)

func TestAllProfiles(t *testing.T) {
	profiles := AllProfiles()

	if len(profiles) != 6 {
		t.Fatalf("expected 6 profiles, got %d", len(profiles))
	}

	expectedIDs := []string{
		"competitive_fps",
		"open_world",
		"moba_strategy",
		"racing_sim",
		"casual",
		"nuclear",
	}

	for i, expectedID := range expectedIDs {
		if profiles[i].ID != expectedID {
			t.Errorf("profile[%d]: expected ID %q, got %q", i, expectedID, profiles[i].ID)
		}
	}
}

func TestGetProfileByID(t *testing.T) {
	t.Run("ValidIDs", func(t *testing.T) {
		validIDs := []string{
			"competitive_fps",
			"open_world",
			"moba_strategy",
			"racing_sim",
			"casual",
			"nuclear",
		}

		for _, id := range validIDs {
			profile := GetProfileByID(id)
			if profile == nil {
				t.Errorf("GetProfileByID(%q) returned nil, expected a profile", id)
				continue
			}
			if profile.ID != id {
				t.Errorf("GetProfileByID(%q) returned profile with ID %q", id, profile.ID)
			}
		}
	})

	t.Run("InvalidID", func(t *testing.T) {
		profile := GetProfileByID("nonexistent_profile")
		if profile != nil {
			t.Errorf("GetProfileByID with invalid ID should return nil, got %+v", profile)
		}
	})

	t.Run("EmptyID", func(t *testing.T) {
		profile := GetProfileByID("")
		if profile != nil {
			t.Errorf("GetProfileByID with empty ID should return nil, got %+v", profile)
		}
	})
}

func TestNuclearProfileHasAllTweaks(t *testing.T) {
	nuclear := Nuclear()

	// The nuclear profile should have every tweak enabled
	allTweakIDs := []string{
		"mouse_raw_input",
		"mouse_disable_acceleration",
		"gpu_low_latency",
		"gpu_max_performance",
		"kill_bloatware",
		"timer_resolution",
		"disable_game_dvr",
		"disable_game_bar",
		"disable_game_mode",
		"disable_nagle",
		"disable_fullscreen_optimize",
		"keyboard_repeat_max",
		"core_parking_off",
		"disable_indexing",
		"disable_hpet",
		"ultimate_power_plan",
		"disable_sysmain",
		"dns_optimize",
		"flush_network",
		"cpu_priority_high",
		"disable_smooth_scrolling",
		"disable_sticky_keys",
		"disable_filter_keys",
		"disable_toggle_keys",
	}

	for _, tweakID := range allTweakIDs {
		enabled, exists := nuclear.Tweaks[tweakID]
		if !exists {
			t.Errorf("nuclear profile missing tweak %q", tweakID)
			continue
		}
		if !enabled {
			t.Errorf("nuclear profile has tweak %q disabled; expected all tweaks enabled", tweakID)
		}
	}

	// Verify the total count matches
	if len(nuclear.Tweaks) != len(allTweakIDs) {
		t.Errorf("nuclear profile has %d tweaks, expected %d", len(nuclear.Tweaks), len(allTweakIDs))
	}
}

func TestProfileNames(t *testing.T) {
	expectedNames := map[string]string{
		"competitive_fps": "Competitive FPS",
		"open_world":      "Open World",
		"moba_strategy":   "MOBA / Strategy",
		"racing_sim":      "Racing / Sim",
		"casual":          "Casual / Indie",
		"nuclear":         "Nuclear Mode",
	}

	profiles := AllProfiles()
	for _, p := range profiles {
		expectedName, ok := expectedNames[p.ID]
		if !ok {
			t.Errorf("unexpected profile ID: %q", p.ID)
			continue
		}
		if p.Name != expectedName {
			t.Errorf("profile %q: expected name %q, got %q", p.ID, expectedName, p.Name)
		}
	}
}

func TestProfileFields(t *testing.T) {
	profiles := AllProfiles()

	for _, p := range profiles {
		t.Run(p.ID, func(t *testing.T) {
			if p.ID == "" {
				t.Error("profile has empty ID")
			}
			if p.Name == "" {
				t.Error("profile has empty Name")
			}
			if p.Icon == "" {
				t.Error("profile has empty Icon")
			}
			if p.Description == "" {
				t.Error("profile has empty Description")
			}
			if len(p.Tweaks) == 0 {
				t.Error("profile has no tweaks")
			}
		})
	}
}

func TestProfileTweaksAreAllEnabled(t *testing.T) {
	// In each profile, all listed tweaks should be set to true
	profiles := AllProfiles()
	for _, p := range profiles {
		t.Run(p.ID, func(t *testing.T) {
			for tweakID, enabled := range p.Tweaks {
				if !enabled {
					t.Errorf("tweak %q in profile %q is set to false; profiles should only list enabled tweaks", tweakID, p.ID)
				}
			}
		})
	}
}

func TestCompetitiveFPSProfile(t *testing.T) {
	fps := CompetitiveFPS()

	if fps.ID != "competitive_fps" {
		t.Errorf("expected ID %q, got %q", "competitive_fps", fps.ID)
	}

	// Competitive FPS should have mouse and network tweaks
	requiredTweaks := []string{
		"mouse_raw_input",
		"mouse_disable_acceleration",
		"disable_nagle",
		"disable_game_dvr",
	}
	for _, tw := range requiredTweaks {
		if !fps.Tweaks[tw] {
			t.Errorf("competitive_fps missing required tweak: %q", tw)
		}
	}
}

func TestCasualProfile(t *testing.T) {
	casual := Casual()

	if casual.ID != "casual" {
		t.Errorf("expected ID %q, got %q", "casual", casual.ID)
	}

	// Casual should have fewer tweaks than nuclear
	nuclear := Nuclear()
	if len(casual.Tweaks) >= len(nuclear.Tweaks) {
		t.Errorf("casual profile should have fewer tweaks than nuclear: casual=%d, nuclear=%d",
			len(casual.Tweaks), len(nuclear.Tweaks))
	}
}
