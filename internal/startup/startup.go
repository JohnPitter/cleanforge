package startup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// ---------- Types ----------

// StartupItem represents a single program configured to run at Windows boot.
type StartupItem struct {
	Name          string `json:"name"`
	Path          string `json:"path"`
	Publisher     string `json:"publisher"`
	Impact        string `json:"impact"`    // "high", "medium", "low", "unknown"
	Enabled       bool   `json:"enabled"`
	Location      string `json:"location"` // "registry_hkcu", "registry_hklm", "startup_folder", "task_scheduler"
	RegistryKey   string `json:"-"`
	RegistryValue string `json:"-"`
}

// ---------- Known impact map ----------

// knownImpact maps executable names (lowercase) to their startup impact.
var knownImpact = map[string]string{
	"teams.exe":                 "high",
	"msteams.exe":               "high",
	"onedrive.exe":              "high",
	"spotify.exe":               "medium",
	"discord.exe":               "medium",
	"slack.exe":                 "high",
	"skype.exe":                 "medium",
	"skypeapp.exe":              "medium",
	"steam.exe":                 "medium",
	"epicgameslauncher.exe":     "medium",
	"googledrivesync.exe":       "high",
	"dropbox.exe":               "high",
	"adobearm.exe":              "low",
	"ccleaner.exe":              "low",
	"itunes.exe":                "high",
	"ituneshelper.exe":          "medium",
	"cortana.exe":               "medium",
	"searchui.exe":              "medium",
	"yourphone.exe":             "medium",
	"phoneexperiencehost.exe":   "medium",
	"gamebar.exe":               "low",
	"gamebarpresencewriter.exe": "low",
	"msedge.exe":                "high",
	"chrome.exe":                "high",
	"firefox.exe":               "high",
	"brave.exe":                 "high",
	"opera.exe":                 "high",
	"zoom.exe":                  "medium",
	"webex.exe":                 "medium",
	"vmware-tray.exe":           "medium",
	"virtualbox.exe":            "medium",
	"nordvpn.exe":               "medium",
	"expressvpn.exe":            "medium",
	"razer synapse.exe":         "medium",
	"razersynapse.exe":          "medium",
	"logitechg.exe":             "medium",
	"icue.exe":                  "high",
	"corsair.exe":               "high",
	"wallpaperengine.exe":       "high",
	"nvidia share.exe":          "medium",
	"nvbackend.exe":             "medium",
	"nvcpl.dll":                 "low",
	"jusched.exe":               "low",
	"realtekhdaudiomanager.exe": "low",
}

// ---------- Disabled subkey name ----------

const disabledSubkey = `CleanForge_Disabled`

// ---------- StartupManager ----------

// StartupManager reads and manages Windows startup items.
type StartupManager struct{}

// NewStartupManager creates a new StartupManager instance.
func NewStartupManager() *StartupManager {
	return &StartupManager{}
}

// ---------- Public API ----------

// GetStartupItems collects startup items from all sources.
func (m *StartupManager) GetStartupItems() ([]StartupItem, error) {
	var items []StartupItem

	// 1. HKCU Run
	hkcuItems, err := m.readRegistryRun(registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows\CurrentVersion\Run`, "registry_hkcu", true)
	if err == nil {
		items = append(items, hkcuItems...)
	}

	// HKCU disabled items
	hkcuDisabled, err := m.readRegistryRun(registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows\CurrentVersion\Run\`+disabledSubkey, "registry_hkcu", false)
	if err == nil {
		items = append(items, hkcuDisabled...)
	}

	// 2. HKLM Run
	hklmItems, err := m.readRegistryRun(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Run`, "registry_hklm", true)
	if err == nil {
		items = append(items, hklmItems...)
	}

	// HKLM disabled items
	hklmDisabled, err := m.readRegistryRun(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Run\`+disabledSubkey, "registry_hklm", false)
	if err == nil {
		items = append(items, hklmDisabled...)
	}

	// 3. Shell startup folder
	folderItems, err := m.readStartupFolder()
	if err == nil {
		items = append(items, folderItems...)
	}

	// 4. Task Scheduler disabled startup items
	taskItems, err := m.readTaskSchedulerStartup()
	if err == nil {
		items = append(items, taskItems...)
	}

	return items, nil
}

// DisableStartupItem disables a startup item by moving it to a disabled subkey or renaming it.
func (m *StartupManager) DisableStartupItem(item StartupItem) error {
	switch item.Location {
	case "registry_hkcu":
		return m.disableRegistryItem(registry.CURRENT_USER, item)
	case "registry_hklm":
		return m.disableRegistryItem(registry.LOCAL_MACHINE, item)
	case "startup_folder":
		return m.disableStartupFolderItem(item)
	case "task_scheduler":
		return m.disableScheduledTask(item)
	default:
		return fmt.Errorf("unknown location: %s", item.Location)
	}
}

// EnableStartupItem re-enables a previously disabled startup item.
func (m *StartupManager) EnableStartupItem(item StartupItem) error {
	switch item.Location {
	case "registry_hkcu":
		return m.enableRegistryItem(registry.CURRENT_USER, item)
	case "registry_hklm":
		return m.enableRegistryItem(registry.LOCAL_MACHINE, item)
	case "startup_folder":
		return m.enableStartupFolderItem(item)
	case "task_scheduler":
		return m.enableScheduledTask(item)
	default:
		return fmt.Errorf("unknown location: %s", item.Location)
	}
}

// EstimateImpact returns an impact rating based on the executable name and file size.
func (m *StartupManager) EstimateImpact(path string) string {
	if path == "" {
		return "unknown"
	}

	// Extract executable name from path (may include arguments)
	exePath := extractExePath(path)
	exeName := strings.ToLower(filepath.Base(exePath))

	// Check known impact map
	if impact, ok := knownImpact[exeName]; ok {
		return impact
	}

	// Estimate by file size
	info, err := os.Stat(exePath)
	if err != nil {
		return "unknown"
	}

	sizeMB := float64(info.Size()) / (1024 * 1024)
	switch {
	case sizeMB > 50:
		return "high"
	case sizeMB > 10:
		return "medium"
	case sizeMB > 1:
		return "low"
	default:
		return "low"
	}
}

// ---------- Registry reading ----------

func (m *StartupManager) readRegistryRun(root registry.Key, keyPath, location string, enabled bool) ([]StartupItem, error) {
	key, err := registry.OpenKey(root, keyPath, registry.QUERY_VALUE)
	if err != nil {
		return nil, err
	}
	defer key.Close()

	names, err := key.ReadValueNames(-1)
	if err != nil {
		return nil, err
	}

	var items []StartupItem
	for _, name := range names {
		val, _, verr := key.GetStringValue(name)
		if verr != nil {
			continue
		}

		item := StartupItem{
			Name:          name,
			Path:          val,
			Publisher:      extractPublisher(val),
			Impact:        m.EstimateImpact(val),
			Enabled:       enabled,
			Location:      location,
			RegistryKey:   keyPath,
			RegistryValue: name,
		}
		items = append(items, item)
	}

	return items, nil
}

// ---------- Startup folder reading ----------

func (m *StartupManager) readStartupFolder() ([]StartupItem, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	startupDir := filepath.Join(home, `AppData\Roaming\Microsoft\Windows\Start Menu\Programs\Startup`)
	entries, err := os.ReadDir(startupDir)
	if err != nil {
		return nil, err
	}

	var items []StartupItem
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		fullPath := filepath.Join(startupDir, name)
		enabled := !strings.HasSuffix(name, ".disabled")

		displayName := name
		if !enabled {
			displayName = strings.TrimSuffix(name, ".disabled")
		}

		item := StartupItem{
			Name:      displayName,
			Path:      fullPath,
			Publisher: extractPublisher(fullPath),
			Impact:    m.EstimateImpact(fullPath),
			Enabled:   enabled,
			Location:  "startup_folder",
		}
		items = append(items, item)
	}

	return items, nil
}

// ---------- Task Scheduler reading ----------

func (m *StartupManager) readTaskSchedulerStartup() ([]StartupItem, error) {
	out, err := exec.Command("schtasks", "/query", "/fo", "CSV", "/nh").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("schtasks failed: %w", err)
	}

	var items []StartupItem
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// CSV format: "TaskName","Next Run Time","Status"
		parts := parseCSVLine(line)
		if len(parts) < 3 {
			continue
		}

		taskName := parts[0]
		// Only include tasks that look like startup-related items
		if !isStartupTask(taskName) {
			continue
		}

		status := strings.TrimSpace(parts[2])
		enabled := !strings.EqualFold(status, "Disabled")

		item := StartupItem{
			Name:     filepath.Base(taskName),
			Path:     taskName,
			Impact:   "unknown",
			Enabled:  enabled,
			Location: "task_scheduler",
		}
		items = append(items, item)
	}

	return items, nil
}

// isStartupTask checks if a scheduled task is likely a startup item.
func isStartupTask(taskName string) bool {
	lower := strings.ToLower(taskName)
	startupKeywords := []string{
		"startup", "logon", "boot", "autostart",
		"update", "updater", "helper",
		"google", "adobe", "microsoft", "mozilla",
		"brave", "opera", "spotify", "discord", "steam",
	}
	for _, keyword := range startupKeywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

// ---------- Disable / Enable registry ----------

func (m *StartupManager) disableRegistryItem(root registry.Key, item StartupItem) error {
	if !item.Enabled {
		return nil
	}

	// Read the current value
	srcKey, err := registry.OpenKey(root, item.RegistryKey, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open source key: %w", err)
	}

	val, _, err := srcKey.GetStringValue(item.RegistryValue)
	srcKey.Close()
	if err != nil {
		return fmt.Errorf("read value %s: %w", item.RegistryValue, err)
	}

	// Write to the disabled subkey
	disabledPath := item.RegistryKey + `\` + disabledSubkey
	dstKey, _, err := registry.CreateKey(root, disabledPath, registry.ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("create disabled key: %w", err)
	}
	if err := dstKey.SetStringValue(item.RegistryValue, val); err != nil {
		dstKey.Close()
		return fmt.Errorf("write to disabled key: %w", err)
	}
	dstKey.Close()

	// Delete from the original key
	srcKey2, err := registry.OpenKey(root, item.RegistryKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("reopen source key: %w", err)
	}
	defer srcKey2.Close()

	return srcKey2.DeleteValue(item.RegistryValue)
}

func (m *StartupManager) enableRegistryItem(root registry.Key, item StartupItem) error {
	if item.Enabled {
		return nil
	}

	// Read from the disabled subkey
	disabledPath := item.RegistryKey
	// If the RegistryKey already points to the disabled subkey, use it directly;
	// otherwise, construct the disabled path.
	if !strings.HasSuffix(item.RegistryKey, disabledSubkey) {
		disabledPath = item.RegistryKey + `\` + disabledSubkey
	}

	srcKey, err := registry.OpenKey(root, disabledPath, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open disabled key: %w", err)
	}

	val, _, err := srcKey.GetStringValue(item.RegistryValue)
	if err != nil {
		srcKey.Close()
		return fmt.Errorf("read disabled value: %w", err)
	}

	// Delete from disabled
	_ = srcKey.DeleteValue(item.RegistryValue)
	srcKey.Close()

	// Determine the original (enabled) key path
	enabledPath := item.RegistryKey
	if strings.HasSuffix(enabledPath, `\`+disabledSubkey) {
		enabledPath = strings.TrimSuffix(enabledPath, `\`+disabledSubkey)
	}

	// Write back to the original key
	dstKey, _, err := registry.CreateKey(root, enabledPath, registry.ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("open original key: %w", err)
	}
	defer dstKey.Close()

	return dstKey.SetStringValue(item.RegistryValue, val)
}

// ---------- Disable / Enable startup folder ----------

func (m *StartupManager) disableStartupFolderItem(item StartupItem) error {
	if !item.Enabled {
		return nil
	}
	newPath := item.Path + ".disabled"
	return os.Rename(item.Path, newPath)
}

func (m *StartupManager) enableStartupFolderItem(item StartupItem) error {
	if item.Enabled {
		return nil
	}
	// The stored path includes ".disabled" suffix for disabled items
	originalPath := item.Path
	if !strings.HasSuffix(originalPath, ".disabled") {
		originalPath = originalPath + ".disabled"
	}
	newPath := strings.TrimSuffix(originalPath, ".disabled")
	return os.Rename(originalPath, newPath)
}

// ---------- Disable / Enable task scheduler ----------

func (m *StartupManager) disableScheduledTask(item StartupItem) error {
	return exec.Command("schtasks", "/Change", "/TN", item.Path, "/Disable").Run()
}

func (m *StartupManager) enableScheduledTask(item StartupItem) error {
	return exec.Command("schtasks", "/Change", "/TN", item.Path, "/Enable").Run()
}

// ---------- Helpers ----------

// extractExePath extracts the executable path from a string that may include arguments.
func extractExePath(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	// Handle quoted paths: "C:\Program Files\app.exe" -args
	if raw[0] == '"' {
		end := strings.Index(raw[1:], `"`)
		if end >= 0 {
			return raw[1 : end+1]
		}
		return strings.Trim(raw, `"`)
	}

	// Unquoted: find .exe boundary
	lower := strings.ToLower(raw)
	if idx := strings.Index(lower, ".exe"); idx >= 0 {
		return raw[:idx+4]
	}

	// Fallback: first space-separated token
	parts := strings.Fields(raw)
	if len(parts) > 0 {
		return parts[0]
	}

	return raw
}

// extractPublisher tries to derive a publisher name from the executable path.
func extractPublisher(rawPath string) string {
	exePath := extractExePath(rawPath)
	lower := strings.ToLower(exePath)

	publishers := map[string]string{
		"microsoft":  "Microsoft Corporation",
		"google":     "Google LLC",
		"adobe":      "Adobe Inc.",
		"mozilla":    "Mozilla Foundation",
		"valve":      "Valve Corporation",
		"steam":      "Valve Corporation",
		"epic games": "Epic Games Inc.",
		"discord":    "Discord Inc.",
		"spotify":    "Spotify AB",
		"slack":      "Salesforce (Slack)",
		"zoom":       "Zoom Video Communications",
		"nvidia":     "NVIDIA Corporation",
		"amd":        "AMD Inc.",
		"intel":      "Intel Corporation",
		"realtek":    "Realtek Semiconductor",
		"logitech":   "Logitech International",
		"razer":      "Razer Inc.",
		"corsair":    "Corsair Components",
		"brave":      "Brave Software",
		"opera":      "Opera Software",
		"dropbox":    "Dropbox Inc.",
		"onedrive":   "Microsoft Corporation",
	}

	for keyword, publisher := range publishers {
		if strings.Contains(lower, keyword) {
			return publisher
		}
	}

	return ""
}

// parseCSVLine splits a simple CSV line respecting double-quote fields.
func parseCSVLine(line string) []string {
	var fields []string
	var current strings.Builder
	inQuotes := false

	for _, r := range line {
		switch {
		case r == '"':
			inQuotes = !inQuotes
		case r == ',' && !inQuotes:
			fields = append(fields, current.String())
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}
	fields = append(fields, current.String())

	return fields
}
