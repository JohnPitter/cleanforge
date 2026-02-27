//go:build e2e

package e2e_test

import (
	"net"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"cleanforge/internal/backup"
	"cleanforge/internal/cleaner"
	"cleanforge/internal/gaming"
	"cleanforge/internal/gaming/profiles"
	"cleanforge/internal/memory"
	"cleanforge/internal/monitor"
	"cleanforge/internal/network"
	"cleanforge/internal/privacy"
	"cleanforge/internal/startup"
	"cleanforge/internal/system"
	"cleanforge/internal/toolkit"
)

// --- System Info ---

// TestE2E_SystemInfoFlow exercises the full system info pipeline and verifies
// the health score, CPU model, and disk list are populated.
func TestE2E_SystemInfoFlow(t *testing.T) {
	info, err := system.GetSystemInfo()
	if err != nil {
		t.Fatalf("GetSystemInfo() error: %v", err)
	}

	// Health score must be between 0 and 100
	if info.HealthScore < 0 || info.HealthScore > 100 {
		t.Errorf("HealthScore = %d, want 0-100", info.HealthScore)
	}

	// CPU model should not be empty on a real system
	if info.CPUModel == "" {
		t.Error("CPUModel is empty")
	}

	// There should be at least 1 disk
	if len(info.Disks) < 1 {
		t.Error("expected at least 1 disk, got 0")
	}

	// OS should be "windows"
	if info.OS != "windows" {
		t.Errorf("OS = %q, want %q", info.OS, "windows")
	}

	// Hostname should not be empty
	if info.Hostname == "" {
		t.Error("Hostname is empty")
	}

	// CPUCores should be at least 1
	if info.CPUCores < 1 {
		t.Errorf("CPUCores = %d, want >= 1", info.CPUCores)
	}

	// CPUThreads should be >= CPUCores
	if info.CPUThreads < info.CPUCores {
		t.Errorf("CPUThreads (%d) < CPUCores (%d)", info.CPUThreads, info.CPUCores)
	}

	// RAMTotal should be > 0
	if info.RAMTotal == 0 {
		t.Error("RAMTotal is 0")
	}

	// RAMUsage should be between 0 and 100
	if info.RAMUsage < 0 || info.RAMUsage > 100 {
		t.Errorf("RAMUsage = %f, want 0-100", info.RAMUsage)
	}

	// CPUUsage should be between 0 and 100
	if info.CPUUsage < 0 || info.CPUUsage > 100 {
		t.Errorf("CPUUsage = %f, want 0-100", info.CPUUsage)
	}

	// Uptime should not be empty
	if info.Uptime == "" {
		t.Error("Uptime is empty")
	}

	// Log info for debugging
	t.Logf("CPU: %s (%d cores / %d threads)", info.CPUModel, info.CPUCores, info.CPUThreads)
	t.Logf("RAM: %d MB total, %.1f%% used", info.RAMTotal/(1024*1024), info.RAMUsage)
	t.Logf("GPU: %s (driver: %s)", info.GPUName, info.GPUDriver)
	t.Logf("Health Score: %d", info.HealthScore)
	t.Logf("Uptime: %s", info.Uptime)
}

// --- Cleaner ---

// TestE2E_CleanerFlow creates a Cleaner, runs Scan, and verifies the result
// has categories with expected fields. Does NOT actually clean real system files.
func TestE2E_CleanerFlow(t *testing.T) {
	// Use the current user for real paths (scan only, no clean)
	username := os.Getenv("USERNAME")
	if username == "" {
		t.Skip("USERNAME env var not set")
	}

	c := cleaner.NewCleaner(username)
	result, err := c.Scan()
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	// Should have categories
	if len(result.Categories) == 0 {
		t.Error("Scan returned 0 categories")
	}

	// Total size should be >= 0
	if result.TotalSize < 0 {
		t.Errorf("TotalSize = %d, want >= 0", result.TotalSize)
	}

	// TotalFiles should be >= 0
	if result.TotalFiles < 0 {
		t.Errorf("TotalFiles = %d, want >= 0", result.TotalFiles)
	}

	// Each category should have required fields
	for _, cat := range result.Categories {
		if cat.ID == "" {
			t.Error("category has empty ID")
		}
		if cat.Name == "" {
			t.Errorf("category %q has empty Name", cat.ID)
		}
		if cat.Description == "" {
			t.Errorf("category %q has empty Description", cat.ID)
		}
		if cat.Risk == "" {
			t.Errorf("category %q has empty Risk", cat.ID)
		}
		if cat.Risk != "safe" && cat.Risk != "low" && cat.Risk != "medium" {
			t.Errorf("category %q has invalid Risk: %q", cat.ID, cat.Risk)
		}
		if cat.Size < 0 {
			t.Errorf("category %q has negative Size: %d", cat.ID, cat.Size)
		}
		if cat.FileCount < 0 {
			t.Errorf("category %q has negative FileCount: %d", cat.ID, cat.FileCount)
		}
	}

	// Verify Clean with empty IDs is a no-op
	cleanResult, err := c.Clean([]string{})
	if err != nil {
		t.Errorf("Clean([]) error: %v", err)
	}
	if cleanResult.FreedSpace != 0 {
		t.Errorf("Clean([]) freed %d bytes, want 0", cleanResult.FreedSpace)
	}

	t.Logf("Scan found %d categories, %d files, %d bytes total",
		len(result.Categories), result.TotalFiles, result.TotalSize)
}

// --- Gaming Profiles ---

// TestE2E_GamingProfilesComplete loads all profiles and verifies each has
// required fields and a non-empty tweaks map.
func TestE2E_GamingProfilesComplete(t *testing.T) {
	allProfiles := profiles.AllProfiles()

	if len(allProfiles) == 0 {
		t.Fatal("AllProfiles() returned 0 profiles")
	}

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
		if len(p.Tweaks) == 0 {
			t.Errorf("profile %q has no tweaks", p.ID)
		}

		// Count enabled tweaks
		enabledCount := 0
		for _, enabled := range p.Tweaks {
			if enabled {
				enabledCount++
			}
		}
		if enabledCount == 0 {
			t.Errorf("profile %q has 0 enabled tweaks", p.ID)
		}

		t.Logf("Profile %q: %s - %d enabled tweaks", p.ID, p.Name, enabledCount)
	}

	// Verify GetProfileByID works
	for _, p := range allProfiles {
		found := profiles.GetProfileByID(p.ID)
		if found == nil {
			t.Errorf("GetProfileByID(%q) returned nil", p.ID)
		}
	}

	// Verify unknown ID returns nil
	if profiles.GetProfileByID("this_does_not_exist") != nil {
		t.Error("GetProfileByID for unknown ID should return nil")
	}
}

// --- Network ---

// TestE2E_NetworkPresets gets all DNS presets and verifies the IPs are valid IPv4 addresses.
func TestE2E_NetworkPresets(t *testing.T) {
	presets := network.GetDNSPresets()

	if len(presets) == 0 {
		t.Fatal("GetDNSPresets() returned 0 presets")
	}

	ipv4Re := regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`)

	for _, preset := range presets {
		if !ipv4Re.MatchString(preset.Primary) {
			t.Errorf("preset %q primary %q is not a valid IPv4 pattern", preset.ID, preset.Primary)
		}
		if !ipv4Re.MatchString(preset.Secondary) {
			t.Errorf("preset %q secondary %q is not a valid IPv4 pattern", preset.ID, preset.Secondary)
		}

		// Verify using net.ParseIP
		if net.ParseIP(preset.Primary) == nil {
			t.Errorf("preset %q primary %q fails net.ParseIP", preset.ID, preset.Primary)
		}
		if net.ParseIP(preset.Secondary) == nil {
			t.Errorf("preset %q secondary %q fails net.ParseIP", preset.ID, preset.Secondary)
		}
	}

	t.Logf("Verified %d DNS presets", len(presets))
}

// --- Privacy ---

// TestE2E_PrivacyTweaksComplete gets all tweaks and verifies there are 12,
// each with required fields.
func TestE2E_PrivacyTweaksComplete(t *testing.T) {
	tweaks, err := privacy.GetPrivacyTweaks()
	if err != nil {
		t.Fatalf("GetPrivacyTweaks() error: %v", err)
	}

	expectedCount := 12
	if len(tweaks) != expectedCount {
		t.Errorf("GetPrivacyTweaks returned %d tweaks, want %d", len(tweaks), expectedCount)
	}

	validCategories := map[string]bool{
		"telemetry": true,
		"tracking":  true,
		"ads":       true,
		"cortana":   true,
	}

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
		if !validCategories[tw.Category] {
			t.Errorf("tweak %q has unknown Category: %q", tw.ID, tw.Category)
		}

		// Check for duplicate IDs
		if seenIDs[tw.ID] {
			t.Errorf("duplicate tweak ID: %q", tw.ID)
		}
		seenIDs[tw.ID] = true

		t.Logf("Privacy tweak: %q (%s) - applied: %v", tw.Name, tw.Category, tw.Applied)
	}
}

// --- Memory ---

// TestE2E_MemoryStatus gets memory status and verifies total > used > 0
// and usage is between 0 and 100.
func TestE2E_MemoryStatus(t *testing.T) {
	status, err := memory.GetMemoryStatus()
	if err != nil {
		t.Fatalf("GetMemoryStatus() error: %v", err)
	}

	// Total should be > 0
	if status.Total == 0 {
		t.Error("MemoryStatus.Total is 0")
	}

	// Used should be > 0
	if status.Used == 0 {
		t.Error("MemoryStatus.Used is 0")
	}

	// Used should be <= Total
	if status.Used > status.Total {
		t.Errorf("MemoryStatus.Used (%d) > Total (%d)", status.Used, status.Total)
	}

	// Available should be > 0
	if status.Available == 0 {
		t.Error("MemoryStatus.Available is 0")
	}

	// UsagePercent should be between 0 and 100
	if status.UsagePercent < 0 || status.UsagePercent > 100 {
		t.Errorf("MemoryStatus.UsagePercent = %f, want 0-100", status.UsagePercent)
	}

	t.Logf("Memory: %d MB total, %d MB used, %.1f%% usage, %d MB available",
		status.Total/(1024*1024), status.Used/(1024*1024),
		status.UsagePercent, status.Available/(1024*1024))
	t.Logf("Top processes: %d", len(status.TopProcesses))
}

// --- Monitor ---

// TestE2E_MonitorSnapshot gets a monitor snapshot and verifies CPU and RAM
// usage are within 0-100.
func TestE2E_MonitorSnapshot(t *testing.T) {
	snapshot, err := monitor.GetSnapshot()
	if err != nil {
		t.Fatalf("GetSnapshot() error: %v", err)
	}

	// CPU usage 0-100
	if snapshot.CPUUsage < 0 || snapshot.CPUUsage > 100 {
		t.Errorf("CPUUsage = %f, want 0-100", snapshot.CPUUsage)
	}

	// RAM usage 0-100
	if snapshot.RAMUsage < 0 || snapshot.RAMUsage > 100 {
		t.Errorf("RAMUsage = %f, want 0-100", snapshot.RAMUsage)
	}

	// DiskUsage 0-100
	if snapshot.DiskUsage < 0 || snapshot.DiskUsage > 100 {
		t.Errorf("DiskUsage = %f, want 0-100", snapshot.DiskUsage)
	}

	// Timestamp should be recent (within last minute)
	if snapshot.Timestamp == 0 {
		t.Error("Timestamp is 0")
	}

	// Temperatures may be 0 if sensors are not available
	t.Logf("Snapshot: CPU=%.1f%%, RAM=%.1f%%, Disk=%.1f%%, CPUTemp=%.1f, GPUTemp=%.1f",
		snapshot.CPUUsage, snapshot.RAMUsage, snapshot.DiskUsage,
		snapshot.CPUTemp, snapshot.GPUTemp)
}

// --- Benchmark ---

// TestE2E_Benchmark runs the benchmark suite and verifies all scores are 0-100.
func TestE2E_Benchmark(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping benchmark in short mode")
	}

	result, err := monitor.RunBenchmark()
	if err != nil {
		t.Fatalf("RunBenchmark() error: %v", err)
	}

	// All scores should be 0-100
	if result.CPUScore < 0 || result.CPUScore > 100 {
		t.Errorf("CPUScore = %d, want 0-100", result.CPUScore)
	}
	if result.RAMScore < 0 || result.RAMScore > 100 {
		t.Errorf("RAMScore = %d, want 0-100", result.RAMScore)
	}
	if result.DiskScore < 0 || result.DiskScore > 100 {
		t.Errorf("DiskScore = %d, want 0-100", result.DiskScore)
	}
	if result.OverallScore < 0 || result.OverallScore > 100 {
		t.Errorf("OverallScore = %d, want 0-100", result.OverallScore)
	}

	// Duration should not be empty
	if result.Duration == "" {
		t.Error("Duration is empty")
	}

	t.Logf("Benchmark: CPU=%d, RAM=%d, Disk=%d, Overall=%d, Duration=%s",
		result.CPUScore, result.RAMScore, result.DiskScore,
		result.OverallScore, result.Duration)
}

// --- Startup ---

// TestE2E_StartupItems gets startup items and verifies the result is a slice
// (may be empty on CI).
func TestE2E_StartupItems(t *testing.T) {
	mgr := startup.NewStartupManager()

	items, err := mgr.GetStartupItems()
	if err != nil {
		// On some CI environments, reading registry/scheduled tasks may fail
		t.Logf("GetStartupItems() error (may be expected in CI): %v", err)
	}

	// items may be nil or empty, which is fine
	if items != nil {
		t.Logf("Found %d startup items", len(items))

		for i, item := range items {
			if i >= 5 {
				t.Logf("... and %d more", len(items)-5)
				break
			}
			t.Logf("  %s (enabled=%v, location=%s, impact=%s)",
				item.Name, item.Enabled, item.Location, item.Impact)
		}
	}
}

// --- Toolkit ---

// TestE2E_ToolkitAdmin verifies IsAdmin returns a bool without panicking.
func TestE2E_ToolkitAdmin(t *testing.T) {
	isAdmin := toolkit.IsAdmin()
	// We just verify it doesn't panic and returns a bool
	t.Logf("IsAdmin() = %v", isAdmin)
}

// --- Backup ---

// TestE2E_BackupPath verifies the backup path is valid and the directory
// can be created.
func TestE2E_BackupPath(t *testing.T) {
	path := backup.GetBackupPath()

	if path == "" {
		t.Fatal("GetBackupPath() returned empty string")
	}

	// Path should be absolute
	if !filepath.IsAbs(path) {
		t.Errorf("GetBackupPath() = %q, want absolute path", path)
	}

	// Directory should exist (GetBackupPath creates it)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("backup path %q does not exist: %v", path, err)
	}
	if !info.IsDir() {
		t.Errorf("backup path %q is not a directory", path)
	}

	// Path should contain ".cleanforge"
	if !containsSubstring(path, ".cleanforge") {
		t.Errorf("backup path %q does not contain '.cleanforge'", path)
	}

	t.Logf("Backup path: %s", path)
}

// TestE2E_BackupHasBackup verifies HasBackup returns a bool without panicking.
func TestE2E_BackupHasBackup(t *testing.T) {
	// Just verify it doesn't panic
	has := backup.HasBackup()
	t.Logf("HasBackup() = %v", has)
}

// --- GPU Detection ---

// TestE2E_GPUDetection detects the GPU and verifies the result has a vendor field.
func TestE2E_GPUDetection(t *testing.T) {
	gb := gaming.NewGameBooster()

	gpu, err := gb.DetectGPU()
	if err != nil {
		// GPU detection may fail in CI/VM environments
		t.Logf("DetectGPU() error (may be expected in CI/VM): %v", err)
		return
	}

	if gpu.Name == "" {
		t.Error("GPUInfo.Name is empty")
	}
	if gpu.Vendor == "" {
		t.Error("GPUInfo.Vendor is empty")
	}

	validVendors := map[string]bool{
		"nvidia":  true,
		"amd":     true,
		"intel":   true,
		"unknown": true,
	}
	if !validVendors[gpu.Vendor] {
		t.Errorf("GPUInfo.Vendor = %q, want one of nvidia/amd/intel/unknown", gpu.Vendor)
	}

	if gpu.ProfileName == "" {
		t.Error("GPUInfo.ProfileName is empty")
	}

	t.Logf("GPU: %s (vendor=%s, driver=%s, profile=%s)",
		gpu.Name, gpu.Vendor, gpu.Driver, gpu.ProfileName)
}

// --- Gaming Tweaks E2E ---

// TestE2E_GamingTweaksList verifies the tweak catalog is accessible and complete.
func TestE2E_GamingTweaksList(t *testing.T) {
	gb := gaming.NewGameBooster()
	tweaks := gb.GetAvailableTweaks()

	if len(tweaks) == 0 {
		t.Fatal("GetAvailableTweaks() returned 0 tweaks")
	}

	// All tweaks should have required fields
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

	t.Logf("Found %d gaming tweaks", len(tweaks))
}

// TestE2E_BoostStatusInitial verifies the initial boost status is inactive.
func TestE2E_BoostStatusInitial(t *testing.T) {
	gb := gaming.NewGameBooster()
	status := gb.GetBoostStatus()

	if status.Active {
		t.Error("initial BoostStatus.Active should be false")
	}
	if status.Profile != "" {
		t.Errorf("initial BoostStatus.Profile = %q, want empty", status.Profile)
	}
	if len(status.TweaksApplied) != 0 {
		t.Errorf("initial BoostStatus.TweaksApplied has %d items, want 0", len(status.TweaksApplied))
	}
}

// --- Helper ---

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
