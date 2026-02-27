package main

import (
	"context"
	"os/user"

	"cleanforge/internal/backup"
	"cleanforge/internal/cleaner"
	"cleanforge/internal/gaming"
	"cleanforge/internal/memory"
	"cleanforge/internal/monitor"
	"cleanforge/internal/network"
	"cleanforge/internal/privacy"
	"cleanforge/internal/startup"
	"cleanforge/internal/system"
	"cleanforge/internal/toolkit"
)

type App struct {
	ctx           context.Context
	username      string
	cleanerModule *cleaner.Cleaner
	gamingModule  *gaming.GameBooster
	startupModule *startup.StartupManager
}

func NewApp() *App {
	u, _ := user.Current()
	username := "User"
	if u != nil {
		username = u.Username
		for i := len(username) - 1; i >= 0; i-- {
			if username[i] == '\\' {
				username = username[i+1:]
				break
			}
		}
	}

	return &App{
		username:      username,
		cleanerModule: cleaner.NewCleaner(username),
		gamingModule:  gaming.NewGameBooster(),
		startupModule: startup.NewStartupManager(),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// ============================================================
// System Info
// ============================================================

func (a *App) GetSystemInfo() (*system.SystemInfo, error) {
	return system.GetSystemInfo()
}

// ============================================================
// Cleaner
// ============================================================

func (a *App) ScanSystem() (*cleaner.ScanResult, error) {
	return a.cleanerModule.Scan()
}

func (a *App) CleanSystem(categoryIDs []string) (*cleaner.CleanResult, error) {
	return a.cleanerModule.Clean(categoryIDs)
}

// ============================================================
// Game Boost
// ============================================================

func (a *App) DetectGPU() (*gaming.GPUInfo, error) {
	return a.gamingModule.DetectGPU()
}

func (a *App) GetBoostStatus() *gaming.BoostStatus {
	return a.gamingModule.GetBoostStatus()
}

func (a *App) GetGameProfiles() []gaming.GameProfile {
	return a.gamingModule.GetProfiles()
}

func (a *App) ApplyGameProfile(profileID string) error {
	return a.gamingModule.ApplyProfile(profileID)
}

func (a *App) RestoreGameSettings() error {
	return a.gamingModule.RestoreAll()
}

func (a *App) GetAvailableTweaks() []gaming.TweakInfo {
	return a.gamingModule.GetAvailableTweaks()
}

// ============================================================
// Startup Manager
// ============================================================

func (a *App) GetStartupItems() ([]startup.StartupItem, error) {
	return a.startupModule.GetStartupItems()
}

func (a *App) DisableStartupItem(item startup.StartupItem) error {
	return a.startupModule.DisableStartupItem(item)
}

func (a *App) EnableStartupItem(item startup.StartupItem) error {
	return a.startupModule.EnableStartupItem(item)
}

// ============================================================
// Network
// ============================================================

func (a *App) GetNetworkStatus() (*network.NetworkStatus, error) {
	return network.GetNetworkStatus()
}

func (a *App) SetDNS(preset network.DNSPreset) error {
	return network.SetDNS(preset)
}

func (a *App) ResetDNS() error {
	return network.ResetDNS()
}

func (a *App) DisableNagle() error {
	return network.DisableNagle()
}

func (a *App) EnableNagle() error {
	return network.EnableNagle()
}

func (a *App) FlushNetwork() (string, error) {
	return network.FlushNetwork()
}

func (a *App) PingTest(host string) (float64, error) {
	return network.PingTest(host)
}

// ============================================================
// Toolkit
// ============================================================

func (a *App) GetIsAdmin() bool {
	return toolkit.IsAdmin()
}

func (a *App) RunSFC() (*toolkit.ToolResult, error) {
	return toolkit.RunSFC()
}

func (a *App) RunDISM() (*toolkit.ToolResult, error) {
	return toolkit.RunDISM()
}

func (a *App) GetBloatwareApps() ([]toolkit.BloatwareApp, error) {
	return toolkit.GetBloatwareApps()
}

func (a *App) RemoveBloatware(packageNames []string) (*toolkit.ToolResult, error) {
	return toolkit.RemoveBloatware(packageNames)
}

func (a *App) RebuildIconCache() (*toolkit.ToolResult, error) {
	return toolkit.RebuildIconCache()
}

func (a *App) RebuildFontCache() (*toolkit.ToolResult, error) {
	return toolkit.RebuildFontCache()
}

func (a *App) ResetWindowsSearch() (*toolkit.ToolResult, error) {
	return toolkit.ResetWindowsSearch()
}

func (a *App) RepairWindowsUpdate() (*toolkit.ToolResult, error) {
	return toolkit.RepairWindowsUpdate()
}

// ============================================================
// Privacy
// ============================================================

func (a *App) GetPrivacyTweaks() ([]privacy.PrivacyTweak, error) {
	return privacy.GetPrivacyTweaks()
}

func (a *App) TogglePrivacyTweak(tweakID string) error {
	tweaks, err := privacy.GetPrivacyTweaks()
	if err != nil {
		return err
	}
	for _, t := range tweaks {
		if t.ID == tweakID {
			if t.Applied {
				return privacy.RestoreAll()
			}
			return privacy.ApplyTweak(tweakID)
		}
	}
	return privacy.ApplyTweak(tweakID)
}

func (a *App) ApplyAllPrivacy() error {
	return privacy.ApplyAll()
}

func (a *App) RestoreAllPrivacy() error {
	return privacy.RestoreAll()
}

// ============================================================
// Memory
// ============================================================

func (a *App) GetMemoryStatus() (*memory.MemoryStatus, error) {
	return memory.GetMemoryStatus()
}

func (a *App) FlushMemory() error {
	return memory.FlushStandbyList()
}

// ============================================================
// Monitor
// ============================================================

func (a *App) GetMonitorSnapshot() (*monitor.MonitorSnapshot, error) {
	return monitor.GetSnapshot()
}

func (a *App) RunBenchmark() (*monitor.BenchmarkResult, error) {
	return monitor.RunBenchmark()
}

// ============================================================
// Backup
// ============================================================

func (a *App) HasBackup() bool {
	return backup.HasBackup()
}

func (a *App) RestoreAllBackup() error {
	return backup.RestoreAll()
}
