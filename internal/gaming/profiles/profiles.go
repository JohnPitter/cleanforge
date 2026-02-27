package profiles

// GameProfile represents a predefined set of tweaks targeting a specific gaming genre.
type GameProfile struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Icon        string          `json:"icon"`
	Description string          `json:"description"`
	Tweaks      map[string]bool `json:"tweaks"`
}

// AllProfiles returns the complete list of predefined game profiles.
func AllProfiles() []GameProfile {
	return []GameProfile{
		CompetitiveFPS(),
		OpenWorld(),
		MOBAStrategy(),
		RacingSim(),
		Casual(),
		Nuclear(),
	}
}

// GetProfileByID returns a profile by its ID, or nil if not found.
func GetProfileByID(id string) *GameProfile {
	for _, p := range AllProfiles() {
		if p.ID == id {
			return &p
		}
	}
	return nil
}

// CompetitiveFPS returns a profile tuned for competitive shooters like Valorant, CS2, and Apex Legends.
func CompetitiveFPS() GameProfile {
	return GameProfile{
		ID:          "competitive_fps",
		Name:        "Competitive FPS",
		Icon:        "crosshair",
		Description: "Maximum responsiveness for Valorant, CS2, Apex Legends. Focuses on input latency, raw mouse input, and network optimization.",
		Tweaks: map[string]bool{
			"mouse_raw_input":              true,
			"mouse_disable_acceleration":   true,
			"gpu_low_latency":              true,
			"kill_bloatware":               true,
			"timer_resolution":             true,
			"disable_game_dvr":             true,
			"disable_game_bar":             true,
			"disable_nagle":                true,
			"disable_fullscreen_optimize":  true,
			"keyboard_repeat_max":          true,
			"disable_game_mode":            true,
		},
	}
}

// OpenWorld returns a profile tuned for open-world games like Cyberpunk 2077 and GTA V.
func OpenWorld() GameProfile {
	return GameProfile{
		ID:          "open_world",
		Name:        "Open World",
		Icon:        "globe",
		Description: "Maximum sustained performance for Cyberpunk 2077, GTA V, Elden Ring. Focuses on GPU power, CPU unparking, and memory optimization.",
		Tweaks: map[string]bool{
			"gpu_max_performance":          true,
			"core_parking_off":             true,
			"disable_indexing":             true,
			"disable_hpet":                 true,
			"ultimate_power_plan":          true,
			"disable_sysmain":              true,
			"disable_game_dvr":             true,
			"disable_game_bar":             true,
			"disable_fullscreen_optimize":  true,
			"kill_bloatware":               true,
		},
	}
}

// MOBAStrategy returns a profile tuned for MOBAs and strategy games like League of Legends and Dota 2.
func MOBAStrategy() GameProfile {
	return GameProfile{
		ID:          "moba_strategy",
		Name:        "MOBA / Strategy",
		Icon:        "chess",
		Description: "Network and input optimization for League of Legends, Dota 2, Starcraft. Focuses on low network latency and fast key repeat.",
		Tweaks: map[string]bool{
			"disable_nagle":       true,
			"keyboard_repeat_max": true,
			"dns_optimize":        true,
			"flush_network":       true,
			"cpu_priority_high":   true,
			"disable_game_dvr":    true,
			"disable_game_bar":    true,
			"kill_bloatware":      true,
		},
	}
}

// RacingSim returns a profile tuned for racing and simulation games like Forza and F1.
func RacingSim() GameProfile {
	return GameProfile{
		ID:          "racing_sim",
		Name:        "Racing / Sim",
		Icon:        "car",
		Description: "Sustained high FPS for Forza Horizon, F1, Assetto Corsa. Focuses on GPU power, power plan, and display optimization.",
		Tweaks: map[string]bool{
			"gpu_max_performance":          true,
			"disable_fullscreen_optimize":  true,
			"ultimate_power_plan":          true,
			"core_parking_off":             true,
			"disable_game_dvr":             true,
			"disable_game_bar":             true,
			"disable_game_mode":            true,
			"timer_resolution":             true,
			"kill_bloatware":               true,
		},
	}
}

// Casual returns a lightweight profile for casual and indie games like Minecraft.
func Casual() GameProfile {
	return GameProfile{
		ID:          "casual",
		Name:        "Casual / Indie",
		Icon:        "gamepad",
		Description: "Light optimization for Minecraft, Stardew Valley, indie titles. Only kills heavy bloatware and does basic cleanup.",
		Tweaks: map[string]bool{
			"kill_bloatware":    true,
			"disable_game_dvr":  true,
			"disable_game_bar":  true,
			"disable_sysmain":   true,
		},
	}
}

// Nuclear returns the most aggressive profile with every single tweak enabled.
func Nuclear() GameProfile {
	return GameProfile{
		ID:          "nuclear",
		Name:        "Nuclear Mode",
		Icon:        "radiation",
		Description: "EVERYTHING maxed out. Every tweak applied. Use at your own risk. Best for dedicated gaming sessions.",
		Tweaks: map[string]bool{
			"mouse_raw_input":              true,
			"mouse_disable_acceleration":   true,
			"gpu_low_latency":              true,
			"gpu_max_performance":          true,
			"kill_bloatware":               true,
			"timer_resolution":             true,
			"disable_game_dvr":             true,
			"disable_game_bar":             true,
			"disable_game_mode":            true,
			"disable_nagle":                true,
			"disable_fullscreen_optimize":  true,
			"keyboard_repeat_max":          true,
			"core_parking_off":             true,
			"disable_indexing":             true,
			"disable_hpet":                 true,
			"ultimate_power_plan":          true,
			"disable_sysmain":              true,
			"dns_optimize":                 true,
			"flush_network":                true,
			"cpu_priority_high":            true,
			"disable_smooth_scrolling":     true,
			"disable_sticky_keys":          true,
			"disable_filter_keys":          true,
			"disable_toggle_keys":          true,
		},
	}
}
