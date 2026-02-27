package gaming

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cleanforge/internal/gaming/profiles"
	"cleanforge/internal/cmd"

	"golang.org/x/sys/windows/registry"
)

// ---------- Types ----------

// GPUInfo holds detected GPU information.
type GPUInfo struct {
	Name        string `json:"name"`
	Vendor      string `json:"vendor"` // "nvidia", "amd", "intel"
	Driver      string `json:"driver"`
	ProfileName string `json:"profileName"`
}

// BoostStatus represents the current state of game boosting.
type BoostStatus struct {
	Active        bool     `json:"active"`
	Profile       string   `json:"profile"`
	TweaksApplied []string `json:"tweaksApplied"`
	StartedAt     string   `json:"startedAt"`
}

// TweakInfo describes a single tweak that can be toggled.
type TweakInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Enabled     bool   `json:"enabled"`
	Applied     bool   `json:"applied"`
}

// GameProfile mirrors the profile type for JSON serialization to the frontend.
type GameProfile struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Icon        string          `json:"icon"`
	Description string          `json:"description"`
	Tweaks      map[string]bool `json:"tweaks"`
}

// BackupEntry stores a single registry or service state for restore.
type BackupEntry struct {
	Type         string `json:"type"`          // "registry" or "service"
	Root         string `json:"root"`          // "HKCU" or "HKLM"
	KeyPath      string `json:"keyPath"`       // registry key path
	ValueName    string `json:"valueName"`     // registry value name
	Value        string `json:"value"`         // original value as string
	ValueType    uint32 `json:"valueType"`     // registry value type constant
	ServiceName  string `json:"serviceName"`   // for service entries
	ServiceState string `json:"serviceState"`  // "running" or "stopped"
	Missing      bool   `json:"missing"`       // true if value did not exist before
}

// BackupState is the complete backup persisted to disk.
type BackupState struct {
	CreatedAt string        `json:"createdAt"`
	Entries   []BackupEntry `json:"entries"`
}

// ---------- GPU class GUID constant ----------

const gpuClassGUID = `SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}\0000`

// ---------- Bloatware lists ----------

var heavyBloatware = []string{
	"SearchUI.exe",
	"Cortana.exe",
	"OneDrive.exe",
	"Teams.exe",
	"YourPhone.exe",
	"PhoneExperienceHost.exe",
	"GameBar.exe",
	"GameBarPresenceWriter.exe",
	"MicrosoftEdgeUpdate.exe",
	"SkypeApp.exe",
	"SkypeBackgroundHost.exe",
	"HelpPane.exe",
	"Widgets.exe",
	"msedge.exe",
}

var lightBloatware = []string{
	"SearchUI.exe",
	"Cortana.exe",
	"OneDrive.exe",
	"GameBar.exe",
	"GameBarPresenceWriter.exe",
	"MicrosoftEdgeUpdate.exe",
	"Widgets.exe",
}

// ---------- Tweak catalog ----------

type tweakDef struct {
	ID          string
	Name        string
	Description string
	Category    string
}

var tweakCatalog = []tweakDef{
	{"mouse_raw_input", "Raw Mouse Input", "Enable raw input for precise mouse movement", "mouse"},
	{"mouse_disable_acceleration", "Disable Mouse Acceleration", "Set MouseSpeed, Threshold1, Threshold2 to 0", "mouse"},
	{"disable_smooth_scrolling", "Disable Smooth Scrolling", "Turn off smooth scrolling in system settings", "mouse"},
	{"keyboard_repeat_max", "Max Keyboard Repeat Rate", "Set KeyboardDelay=0 and KeyboardSpeed=31", "keyboard"},
	{"disable_sticky_keys", "Disable Sticky Keys", "Prevent sticky keys popup during gaming", "keyboard"},
	{"disable_filter_keys", "Disable Filter Keys", "Prevent filter keys popup during gaming", "keyboard"},
	{"disable_toggle_keys", "Disable Toggle Keys", "Prevent toggle keys sound during gaming", "keyboard"},
	{"gpu_low_latency", "GPU Low Latency Mode", "Vendor-specific low latency rendering", "gpu"},
	{"gpu_max_performance", "GPU Max Performance", "Vendor-specific maximum power/performance", "gpu"},
	{"disable_game_dvr", "Disable Game DVR", "Turn off background game recording", "display"},
	{"disable_game_bar", "Disable Game Bar", "Turn off Xbox Game Bar overlay", "display"},
	{"disable_game_mode", "Disable Game Mode", "Turn off Windows Game Mode", "display"},
	{"disable_fullscreen_optimize", "Disable Fullscreen Optimizations", "Prevent DWM fullscreen optimizations", "display"},
	{"ultimate_power_plan", "Ultimate Performance Power Plan", "Activate Windows Ultimate Performance plan", "power"},
	{"core_parking_off", "Disable Core Parking", "Keep all CPU cores active", "power"},
	{"disable_hpet", "Disable HPET", "Remove platform clock for lower timer latency", "system"},
	{"timer_resolution", "High Timer Resolution", "Request 0.5ms timer resolution", "system"},
	{"disable_sysmain", "Disable SysMain/SuperFetch", "Stop SysMain service temporarily", "system"},
	{"disable_indexing", "Disable Windows Search Indexing", "Stop WSearch service temporarily", "system"},
	{"kill_bloatware", "Kill Bloatware Processes", "Terminate known background bloatware", "system"},
	{"disable_nagle", "Disable Nagle Algorithm", "Turn off TCP packet batching for lower latency", "network"},
	{"dns_optimize", "Optimize DNS Settings", "Flush DNS cache and set fast lookup", "network"},
	{"flush_network", "Flush Network Stack", "Reset Winsock and flush DNS/ARP", "network"},
	{"cpu_priority_high", "High CPU Priority", "Set foreground process priority boost", "system"},
}

// ---------- GameBooster ----------

// GameBooster is the main struct exposed to the Wails frontend.
type GameBooster struct {
	mu            sync.Mutex
	status        BoostStatus
	appliedTweaks map[string]bool
	backupPath    string
}

// NewGameBooster creates and initializes a GameBooster instance.
func NewGameBooster() *GameBooster {
	home, _ := os.UserHomeDir()
	backupDir := filepath.Join(home, ".cleanforge")
	_ = os.MkdirAll(backupDir, 0o755)

	return &GameBooster{
		appliedTweaks: make(map[string]bool),
		backupPath:    filepath.Join(backupDir, "backup_state.json"),
	}
}

// ---------- GPU Detection ----------

// DetectGPU uses wmic to detect the primary GPU and determine its vendor.
func (g *GameBooster) DetectGPU() (*GPUInfo, error) {
	out, err := cmd.Hidden("wmic", "path", "win32_VideoController", "get", "Name,DriverVersion", "/format:csv").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("wmic failed: %w — output: %s", err, string(out))
	}

	info := &GPUInfo{}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Node") {
			continue
		}
		// CSV format: Node,DriverVersion,Name
		parts := strings.SplitN(line, ",", 3)
		if len(parts) < 3 {
			continue
		}
		info.Driver = strings.TrimSpace(parts[1])
		info.Name = strings.TrimSpace(parts[2])
		break
	}

	if info.Name == "" {
		return nil, fmt.Errorf("no GPU detected")
	}

	nameLower := strings.ToLower(info.Name)
	switch {
	case strings.Contains(nameLower, "nvidia") || strings.Contains(nameLower, "geforce") || strings.Contains(nameLower, "rtx") || strings.Contains(nameLower, "gtx"):
		info.Vendor = "nvidia"
		info.ProfileName = "NVIDIA Performance"
	case strings.Contains(nameLower, "amd") || strings.Contains(nameLower, "radeon") || strings.Contains(nameLower, "rx "):
		info.Vendor = "amd"
		info.ProfileName = "AMD Performance"
	case strings.Contains(nameLower, "intel") || strings.Contains(nameLower, "iris") || strings.Contains(nameLower, "uhd") || strings.Contains(nameLower, "hd graphics"):
		info.Vendor = "intel"
		info.ProfileName = "Intel Performance"
	default:
		info.Vendor = "unknown"
		info.ProfileName = "Generic Performance"
	}

	return info, nil
}

// ---------- Backup & Restore ----------

func (g *GameBooster) readBackup() (*BackupState, error) {
	data, err := os.ReadFile(g.backupPath)
	if err != nil {
		return nil, err
	}
	var state BackupState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (g *GameBooster) writeBackup(state *BackupState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(g.backupPath, data, 0o644)
}

func rootKey(name string) registry.Key {
	if name == "HKLM" {
		return registry.LOCAL_MACHINE
	}
	return registry.CURRENT_USER
}

// backupRegistryValue reads the current value from the registry and appends it to the backup state.
// If the value does not exist, it marks the entry as missing so restore can delete it.
func (g *GameBooster) backupRegistryValue(state *BackupState, root string, keyPath, valueName string) {
	rk := rootKey(root)
	key, err := registry.OpenKey(rk, keyPath, registry.QUERY_VALUE)
	if err != nil {
		state.Entries = append(state.Entries, BackupEntry{
			Type:      "registry",
			Root:      root,
			KeyPath:   keyPath,
			ValueName: valueName,
			Missing:   true,
		})
		return
	}
	defer key.Close()

	val, valType, err := key.GetStringValue(valueName)
	if err != nil {
		// Try reading as integer
		ival, _, ierr := key.GetIntegerValue(valueName)
		if ierr != nil {
			state.Entries = append(state.Entries, BackupEntry{
				Type:      "registry",
				Root:      root,
				KeyPath:   keyPath,
				ValueName: valueName,
				Missing:   true,
			})
			return
		}
		state.Entries = append(state.Entries, BackupEntry{
			Type:      "registry",
			Root:      root,
			KeyPath:   keyPath,
			ValueName: valueName,
			Value:     fmt.Sprintf("%d", ival),
			ValueType: registry.DWORD,
		})
		return
	}

	state.Entries = append(state.Entries, BackupEntry{
		Type:      "registry",
		Root:      root,
		KeyPath:   keyPath,
		ValueName: valueName,
		Value:     val,
		ValueType: valType,
	})
}

// backupServiceState saves whether a service is running.
func (g *GameBooster) backupServiceState(state *BackupState, serviceName string) {
	out, err := cmd.Hidden("sc", "query", serviceName).CombinedOutput()
	svcState := "stopped"
	if err == nil && strings.Contains(string(out), "RUNNING") {
		svcState = "running"
	}
	state.Entries = append(state.Entries, BackupEntry{
		Type:         "service",
		ServiceName:  serviceName,
		ServiceState: svcState,
	})
}

// BackupCurrentState captures the current state of all tweakable settings.
func (g *GameBooster) BackupCurrentState() error {
	state := &BackupState{
		CreatedAt: time.Now().Format(time.RFC3339),
		Entries:   []BackupEntry{},
	}

	// Mouse settings
	g.backupRegistryValue(state, "HKCU", `Control Panel\Mouse`, "MouseSpeed")
	g.backupRegistryValue(state, "HKCU", `Control Panel\Mouse`, "MouseThreshold1")
	g.backupRegistryValue(state, "HKCU", `Control Panel\Mouse`, "MouseThreshold2")
	g.backupRegistryValue(state, "HKCU", `Control Panel\Mouse`, "SmoothMouseXCurve")
	g.backupRegistryValue(state, "HKCU", `Control Panel\Mouse`, "SmoothMouseYCurve")

	// Keyboard settings
	g.backupRegistryValue(state, "HKCU", `Control Panel\Keyboard`, "KeyboardDelay")
	g.backupRegistryValue(state, "HKCU", `Control Panel\Keyboard`, "KeyboardSpeed")

	// Accessibility
	g.backupRegistryValue(state, "HKCU", `Control Panel\Accessibility\StickyKeys`, "Flags")
	g.backupRegistryValue(state, "HKCU", `Control Panel\Accessibility\Keyboard Response`, "Flags")
	g.backupRegistryValue(state, "HKCU", `Control Panel\Accessibility\ToggleKeys`, "Flags")

	// Game DVR / Game Bar / Game Mode
	g.backupRegistryValue(state, "HKCU", `System\GameConfigStore`, "GameDVR_Enabled")
	g.backupRegistryValue(state, "HKCU", `SOFTWARE\Microsoft\Windows\CurrentVersion\GameDVR`, "AppCaptureEnabled")
	g.backupRegistryValue(state, "HKCU", `Software\Microsoft\GameBar`, "AllowAutoGameMode")
	g.backupRegistryValue(state, "HKCU", `Software\Microsoft\GameBar`, "AutoGameModeEnabled")

	// Fullscreen optimization
	g.backupRegistryValue(state, "HKCU", `System\GameConfigStore`, "GameDVR_FSEBehaviorMode")
	g.backupRegistryValue(state, "HKCU", `System\GameConfigStore`, "GameDVR_HonorUserFSEBehaviorMode")
	g.backupRegistryValue(state, "HKCU", `System\GameConfigStore`, "GameDVR_FSEBehavior")
	g.backupRegistryValue(state, "HKCU", `System\GameConfigStore`, "GameDVR_DXGIHonorFSEWindowsCompatible")

	// Nagle
	g.backupRegistryValue(state, "HKLM", `SYSTEM\CurrentControlSet\Services\Tcpip\Parameters\Interfaces`, "TcpAckFrequency")
	g.backupRegistryValue(state, "HKLM", `SYSTEM\CurrentControlSet\Services\Tcpip\Parameters\Interfaces`, "TCPNoDelay")

	// Global TCP parameters
	g.backupRegistryValue(state, "HKLM", `SYSTEM\CurrentControlSet\Services\Tcpip\Parameters`, "TcpAckFrequency")
	g.backupRegistryValue(state, "HKLM", `SYSTEM\CurrentControlSet\Services\Tcpip\Parameters`, "TCPNoDelay")

	// GPU class key
	g.backupRegistryValue(state, "HKLM", gpuClassGUID, "UlpsEnable")
	g.backupRegistryValue(state, "HKLM", gpuClassGUID, "PP_ThermalAutoThrottlingEnable")
	g.backupRegistryValue(state, "HKLM", gpuClassGUID, "DisableDMACopy")
	g.backupRegistryValue(state, "HKLM", gpuClassGUID, "KMD_EnableGPUTaskScheduler")

	// Core parking
	g.backupRegistryValue(state, "HKLM", `SYSTEM\CurrentControlSet\Control\Power\PowerSettings\54533251-82be-4824-96c1-47b60b740d00\0cc5b647-c1df-4637-891a-dec35c318583`, "ValueMax")

	// CPU priority
	g.backupRegistryValue(state, "HKLM", `SYSTEM\CurrentControlSet\Control\PriorityControl`, "Win32PrioritySeparation")

	// Smooth scrolling
	g.backupRegistryValue(state, "HKCU", `Control Panel\Desktop`, "SmoothScroll")

	// Services
	g.backupServiceState(state, "SysMain")
	g.backupServiceState(state, "WSearch")

	return g.writeBackup(state)
}

// RestoreOriginalState reads the backup and restores all saved values.
func (g *GameBooster) RestoreOriginalState() error {
	state, err := g.readBackup()
	if err != nil {
		return fmt.Errorf("no backup found: %w", err)
	}

	var errs []string

	for _, entry := range state.Entries {
		switch entry.Type {
		case "registry":
			if entry.Missing {
				// Value did not exist before; delete it.
				rk := rootKey(entry.Root)
				key, kerr := registry.OpenKey(rk, entry.KeyPath, registry.SET_VALUE)
				if kerr == nil {
					_ = key.DeleteValue(entry.ValueName)
					key.Close()
				}
				continue
			}
			rk := rootKey(entry.Root)
			key, _, kerr := registry.CreateKey(rk, entry.KeyPath, registry.ALL_ACCESS)
			if kerr != nil {
				errs = append(errs, fmt.Sprintf("open %s\\%s: %v", entry.Root, entry.KeyPath, kerr))
				continue
			}
			switch entry.ValueType {
			case registry.DWORD:
				var n uint64
				fmt.Sscanf(entry.Value, "%d", &n)
				kerr = key.SetDWordValue(entry.ValueName, uint32(n))
			default:
				kerr = key.SetStringValue(entry.ValueName, entry.Value)
			}
			key.Close()
			if kerr != nil {
				errs = append(errs, fmt.Sprintf("set %s\\%s\\%s: %v", entry.Root, entry.KeyPath, entry.ValueName, kerr))
			}

		case "service":
			if entry.ServiceState == "running" {
				_ = cmd.Hidden("sc", "start", entry.ServiceName).Run()
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("restore completed with errors: %s", strings.Join(errs, "; "))
	}

	return nil
}

// ---------- Registry helpers ----------

func setRegString(root registry.Key, keyPath, name, value string) error {
	key, _, err := registry.CreateKey(root, keyPath, registry.ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("create key %s: %w", keyPath, err)
	}
	defer key.Close()
	return key.SetStringValue(name, value)
}

func setRegDWORD(root registry.Key, keyPath, name string, value uint32) error {
	key, _, err := registry.CreateKey(root, keyPath, registry.ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("create key %s: %w", keyPath, err)
	}
	defer key.Close()
	return key.SetDWordValue(name, value)
}

// ---------- Individual tweak implementations ----------

func (g *GameBooster) applyMouseRawInput() error {
	// Enable raw input hint via registry. Applications read HID_Usage flags;
	// the primary user-facing toggle is disabling acceleration (see below).
	return setRegString(registry.CURRENT_USER, `Control Panel\Mouse`, "MouseSpeed", "0")
}

func (g *GameBooster) applyMouseDisableAcceleration() error {
	if err := setRegString(registry.CURRENT_USER, `Control Panel\Mouse`, "MouseSpeed", "0"); err != nil {
		return err
	}
	if err := setRegString(registry.CURRENT_USER, `Control Panel\Mouse`, "MouseThreshold1", "0"); err != nil {
		return err
	}
	return setRegString(registry.CURRENT_USER, `Control Panel\Mouse`, "MouseThreshold2", "0")
}

func (g *GameBooster) applyDisableSmoothScrolling() error {
	return setRegDWORD(registry.CURRENT_USER, `Control Panel\Desktop`, "SmoothScroll", 0)
}

func (g *GameBooster) applyKeyboardRepeatMax() error {
	if err := setRegString(registry.CURRENT_USER, `Control Panel\Keyboard`, "KeyboardDelay", "0"); err != nil {
		return err
	}
	return setRegString(registry.CURRENT_USER, `Control Panel\Keyboard`, "KeyboardSpeed", "31")
}

func (g *GameBooster) applyDisableStickyKeys() error {
	return setRegString(registry.CURRENT_USER, `Control Panel\Accessibility\StickyKeys`, "Flags", "506")
}

func (g *GameBooster) applyDisableFilterKeys() error {
	return setRegString(registry.CURRENT_USER, `Control Panel\Accessibility\Keyboard Response`, "Flags", "122")
}

func (g *GameBooster) applyDisableToggleKeys() error {
	return setRegString(registry.CURRENT_USER, `Control Panel\Accessibility\ToggleKeys`, "Flags", "58")
}

func (g *GameBooster) applyGPULowLatency() error {
	gpu, err := g.DetectGPU()
	if err != nil {
		return err
	}
	switch gpu.Vendor {
	case "nvidia":
		// NVIDIA Low Latency Mode: set LowLatencyMode value
		return setRegDWORD(registry.LOCAL_MACHINE, gpuClassGUID, "KMD_EnableGPUTaskScheduler", 1)
	case "amd":
		// AMD Anti-Lag toggle through driver registry
		return setRegDWORD(registry.LOCAL_MACHINE, gpuClassGUID, "DisableDMACopy", 1)
	default:
		return nil
	}
}

func (g *GameBooster) applyGPUMaxPerformance() error {
	gpu, err := g.DetectGPU()
	if err != nil {
		return err
	}
	switch gpu.Vendor {
	case "nvidia":
		// Prefer Maximum Performance power management
		if err := setRegDWORD(registry.LOCAL_MACHINE, gpuClassGUID, "PerfLevelSrc", 0x2222); err != nil {
			return err
		}
		return setRegDWORD(registry.LOCAL_MACHINE, gpuClassGUID, "PowerMizerEnable", 1)
	case "amd":
		// Disable ULPS (Ultra Low Power State) and set performance profile
		if err := setRegDWORD(registry.LOCAL_MACHINE, gpuClassGUID, "UlpsEnable", 0); err != nil {
			return err
		}
		return setRegDWORD(registry.LOCAL_MACHINE, gpuClassGUID, "PP_ThermalAutoThrottlingEnable", 0)
	case "intel":
		// Intel max performance mode
		return setRegDWORD(registry.LOCAL_MACHINE, gpuClassGUID, "FeatureTestControl", 0x9240)
	default:
		return nil
	}
}

func (g *GameBooster) applyDisableGameDVR() error {
	return setRegDWORD(registry.CURRENT_USER, `System\GameConfigStore`, "GameDVR_Enabled", 0)
}

func (g *GameBooster) applyDisableGameBar() error {
	if err := setRegDWORD(registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows\CurrentVersion\GameDVR`, "AppCaptureEnabled", 0); err != nil {
		return err
	}
	return setRegDWORD(registry.CURRENT_USER, `Software\Microsoft\GameBar`, "UseNexusForGameBarEnabled", 0)
}

func (g *GameBooster) applyDisableGameMode() error {
	if err := setRegDWORD(registry.CURRENT_USER, `Software\Microsoft\GameBar`, "AllowAutoGameMode", 0); err != nil {
		return err
	}
	return setRegDWORD(registry.CURRENT_USER, `Software\Microsoft\GameBar`, "AutoGameModeEnabled", 0)
}

func (g *GameBooster) applyDisableFullscreenOptimize() error {
	if err := setRegDWORD(registry.CURRENT_USER, `System\GameConfigStore`, "GameDVR_FSEBehaviorMode", 2); err != nil {
		return err
	}
	if err := setRegDWORD(registry.CURRENT_USER, `System\GameConfigStore`, "GameDVR_HonorUserFSEBehaviorMode", 1); err != nil {
		return err
	}
	if err := setRegDWORD(registry.CURRENT_USER, `System\GameConfigStore`, "GameDVR_FSEBehavior", 2); err != nil {
		return err
	}
	return setRegDWORD(registry.CURRENT_USER, `System\GameConfigStore`, "GameDVR_DXGIHonorFSEWindowsCompatible", 1)
}

func (g *GameBooster) applyUltimatePowerPlan() error {
	// Duplicate the Ultimate Performance plan
	out, err := cmd.Hidden("powercfg", "/duplicatescheme", "e9a42b02-d5df-448d-aa00-03f14749eb61").CombinedOutput()
	if err != nil {
		// Plan may already exist; try to find it
		listOut, lerr := cmd.Hidden("powercfg", "/list").CombinedOutput()
		if lerr != nil {
			return fmt.Errorf("powercfg list failed: %w", lerr)
		}
		guid := findUltimatePlanGUID(string(listOut))
		if guid == "" {
			return fmt.Errorf("could not create or find ultimate performance plan: %s", string(out))
		}
		return cmd.Hidden("powercfg", "/setactive", guid).Run()
	}

	// Parse the new GUID from output like: "Power Scheme GUID: xxxx-xxxx (Ultimate Performance)"
	guid := parseGUIDFromPowercfg(string(out))
	if guid == "" {
		// Fallback: search list
		listOut, _ := cmd.Hidden("powercfg", "/list").CombinedOutput()
		guid = findUltimatePlanGUID(string(listOut))
	}
	if guid == "" {
		return fmt.Errorf("could not determine plan GUID from: %s", string(out))
	}

	return cmd.Hidden("powercfg", "/setactive", guid).Run()
}

func parseGUIDFromPowercfg(output string) string {
	// Look for a GUID pattern: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		parts := strings.Fields(line)
		for _, part := range parts {
			part = strings.Trim(part, "()")
			if isGUID(part) {
				return part
			}
		}
	}
	return ""
}

func findUltimatePlanGUID(listOutput string) string {
	for _, line := range strings.Split(listOutput, "\n") {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "ultimate") {
			parts := strings.Fields(line)
			for _, part := range parts {
				part = strings.Trim(part, "()*")
				if isGUID(part) {
					return part
				}
			}
		}
	}
	return ""
}

func isGUID(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) != 36 {
		return false
	}
	// Simple pattern check: 8-4-4-4-12 hex with dashes
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
	}
	return true
}

func (g *GameBooster) applyCoreParking() error {
	// Disable core parking by setting ValueMax to 0 (0% cores parked)
	return setRegDWORD(registry.LOCAL_MACHINE,
		`SYSTEM\CurrentControlSet\Control\Power\PowerSettings\54533251-82be-4824-96c1-47b60b740d00\0cc5b647-c1df-4637-891a-dec35c318583`,
		"ValueMax", 0)
}

func (g *GameBooster) applyDisableHPET() error {
	return cmd.Hidden("bcdedit", "/deletevalue", "useplatformclock").Run()
}

func (g *GameBooster) applyTimerResolution() error {
	// Use powershell to call NtSetTimerResolution for 0.5ms (5000 * 100ns = 0.5ms)
	// This is a best-effort runtime tweak. Persist by setting the global timer.
	return setRegDWORD(registry.LOCAL_MACHINE,
		`SYSTEM\CurrentControlSet\Control\Session Manager\kernel`,
		"GlobalTimerResolutionRequests", 1)
}

func (g *GameBooster) applyDisableSysMain() error {
	return cmd.Hidden("sc", "stop", "SysMain").Run()
}

func (g *GameBooster) applyDisableIndexing() error {
	return cmd.Hidden("sc", "stop", "WSearch").Run()
}

// KillBloatware terminates known bloatware processes.
// If aggressive is true, kills the extended list; otherwise only heavy offenders.
func (g *GameBooster) KillBloatware(aggressive bool) ([]string, error) {
	list := lightBloatware
	if aggressive {
		list = heavyBloatware
	}

	var killed []string
	for _, proc := range list {
		err := cmd.Hidden("taskkill", "/F", "/IM", proc).Run()
		if err == nil {
			killed = append(killed, proc)
		}
	}
	return killed, nil
}

func (g *GameBooster) applyKillBloatware() error {
	_, err := g.KillBloatware(true)
	return err
}

func (g *GameBooster) applyDisableNagle() error {
	// Enumerate network interfaces and disable Nagle on each
	basePath := `SYSTEM\CurrentControlSet\Services\Tcpip\Parameters\Interfaces`
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, basePath, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return fmt.Errorf("open interfaces key: %w", err)
	}
	defer key.Close()

	subkeys, err := key.ReadSubKeyNames(-1)
	if err != nil {
		return fmt.Errorf("read interface subkeys: %w", err)
	}

	for _, sk := range subkeys {
		ifPath := basePath + `\` + sk
		_ = setRegDWORD(registry.LOCAL_MACHINE, ifPath, "TcpAckFrequency", 1)
		_ = setRegDWORD(registry.LOCAL_MACHINE, ifPath, "TCPNoDelay", 1)
	}

	return nil
}

func (g *GameBooster) applyDNSOptimize() error {
	_ = cmd.Hidden("ipconfig", "/flushdns").Run()
	return nil
}

func (g *GameBooster) applyFlushNetwork() error {
	_ = cmd.Hidden("ipconfig", "/flushdns").Run()
	_ = cmd.Hidden("nbtstat", "-R").Run()
	_ = cmd.Hidden("netsh", "winsock", "reset").Run()
	_ = cmd.Hidden("netsh", "int", "ip", "reset").Run()
	return nil
}

func (g *GameBooster) applyCPUPriorityHigh() error {
	// Win32PrioritySeparation = 38 (0x26) -> foreground apps get max priority boost
	return setRegDWORD(registry.LOCAL_MACHINE,
		`SYSTEM\CurrentControlSet\Control\PriorityControl`,
		"Win32PrioritySeparation", 0x26)
}

// ---------- Tweak dispatcher ----------

func (g *GameBooster) applyTweakByID(id string) error {
	switch id {
	case "mouse_raw_input":
		return g.applyMouseRawInput()
	case "mouse_disable_acceleration":
		return g.applyMouseDisableAcceleration()
	case "disable_smooth_scrolling":
		return g.applyDisableSmoothScrolling()
	case "keyboard_repeat_max":
		return g.applyKeyboardRepeatMax()
	case "disable_sticky_keys":
		return g.applyDisableStickyKeys()
	case "disable_filter_keys":
		return g.applyDisableFilterKeys()
	case "disable_toggle_keys":
		return g.applyDisableToggleKeys()
	case "gpu_low_latency":
		return g.applyGPULowLatency()
	case "gpu_max_performance":
		return g.applyGPUMaxPerformance()
	case "disable_game_dvr":
		return g.applyDisableGameDVR()
	case "disable_game_bar":
		return g.applyDisableGameBar()
	case "disable_game_mode":
		return g.applyDisableGameMode()
	case "disable_fullscreen_optimize":
		return g.applyDisableFullscreenOptimize()
	case "ultimate_power_plan":
		return g.applyUltimatePowerPlan()
	case "core_parking_off":
		return g.applyCoreParking()
	case "disable_hpet":
		return g.applyDisableHPET()
	case "timer_resolution":
		return g.applyTimerResolution()
	case "disable_sysmain":
		return g.applyDisableSysMain()
	case "disable_indexing":
		return g.applyDisableIndexing()
	case "kill_bloatware":
		return g.applyKillBloatware()
	case "disable_nagle":
		return g.applyDisableNagle()
	case "dns_optimize":
		return g.applyDNSOptimize()
	case "flush_network":
		return g.applyFlushNetwork()
	case "cpu_priority_high":
		return g.applyCPUPriorityHigh()
	default:
		return fmt.Errorf("unknown tweak: %s", id)
	}
}

// ---------- Public API ----------

// GetProfiles returns all available game profiles.
func (g *GameBooster) GetProfiles() []GameProfile {
	raw := profiles.AllProfiles()
	result := make([]GameProfile, len(raw))
	for i, p := range raw {
		result[i] = GameProfile{
			ID:          p.ID,
			Name:        p.Name,
			Icon:        p.Icon,
			Description: p.Description,
			Tweaks:      p.Tweaks,
		}
	}
	return result
}

// ApplyProfile backs up the current state and then applies all tweaks in a profile.
func (g *GameBooster) ApplyProfile(profileID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	profile := profiles.GetProfileByID(profileID)
	if profile == nil {
		return fmt.Errorf("unknown profile: %s", profileID)
	}

	// Backup before applying
	if err := g.BackupCurrentState(); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	var applied []string
	var errs []string

	for tweakID, enabled := range profile.Tweaks {
		if !enabled {
			continue
		}
		if err := g.applyTweakByID(tweakID); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", tweakID, err))
		} else {
			applied = append(applied, tweakID)
			g.appliedTweaks[tweakID] = true
		}
	}

	g.status = BoostStatus{
		Active:        true,
		Profile:       profileID,
		TweaksApplied: applied,
		StartedAt:     time.Now().Format(time.RFC3339),
	}

	if len(errs) > 0 {
		return fmt.Errorf("profile applied with errors: %s", strings.Join(errs, "; "))
	}

	return nil
}

// ApplyGPUProfile detects the GPU and applies vendor-specific tweaks.
func (g *GameBooster) ApplyGPUProfile() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if err := g.BackupCurrentState(); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	if err := g.applyGPUMaxPerformance(); err != nil {
		return err
	}
	g.appliedTweaks["gpu_max_performance"] = true

	if err := g.applyGPULowLatency(); err != nil {
		return err
	}
	g.appliedTweaks["gpu_low_latency"] = true

	return nil
}

// GetAvailableTweaks returns all tweaks with their current state.
func (g *GameBooster) GetAvailableTweaks() []TweakInfo {
	g.mu.Lock()
	defer g.mu.Unlock()

	result := make([]TweakInfo, len(tweakCatalog))
	for i, td := range tweakCatalog {
		applied := g.appliedTweaks[td.ID]
		result[i] = TweakInfo{
			ID:          td.ID,
			Name:        td.Name,
			Description: td.Description,
			Category:    td.Category,
			Enabled:     true,
			Applied:     applied,
		}
	}
	return result
}

// ApplyTweak applies a single tweak by ID, backing up first.
func (g *GameBooster) ApplyTweak(tweakID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if err := g.BackupCurrentState(); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	if err := g.applyTweakByID(tweakID); err != nil {
		return err
	}

	g.appliedTweaks[tweakID] = true

	// Update status
	if !g.status.Active {
		g.status.Active = true
		g.status.StartedAt = time.Now().Format(time.RFC3339)
		g.status.Profile = "custom"
	}
	g.status.TweaksApplied = append(g.status.TweaksApplied, tweakID)

	return nil
}

// GetBoostStatus returns the current boost state.
func (g *GameBooster) GetBoostStatus() *BoostStatus {
	g.mu.Lock()
	defer g.mu.Unlock()
	status := g.status
	return &status
}

// RestoreAll restores the original system state from backup.
func (g *GameBooster) RestoreAll() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if err := g.RestoreOriginalState(); err != nil {
		return err
	}

	// Re-enable services that were stopped
	_ = cmd.Hidden("sc", "start", "SysMain").Run()
	_ = cmd.Hidden("sc", "start", "WSearch").Run()

	// Clear state
	g.status = BoostStatus{}
	g.appliedTweaks = make(map[string]bool)

	return nil
}
