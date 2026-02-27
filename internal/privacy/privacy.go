package privacy

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// PrivacyTweak represents a single privacy configuration change.
type PrivacyTweak struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"` // "telemetry", "tracking", "ads", "cortana"
	Enabled     bool   `json:"enabled"`  // true = privacy protection is ON (telemetry is disabled)
	Applied     bool   `json:"applied"`
}

// registryTweak describes a registry-based privacy tweak.
type registryTweak struct {
	id          string
	name        string
	description string
	category    string
	entries     []registryEntry
}

// registryEntry describes a single registry value to set.
type registryEntry struct {
	rootKey registry.Key
	path    string
	name    string
	value   uint32
}

// telemetryHosts is the list of Microsoft telemetry domains to block via the hosts file.
var telemetryHosts = []string{
	"vortex.data.microsoft.com",
	"settings-win.data.microsoft.com",
	"watson.telemetry.microsoft.com",
	"watson.microsoft.com",
	"telemetry.microsoft.com",
	"oca.telemetry.microsoft.com",
	"sqm.telemetry.microsoft.com",
	"vortex-win.data.microsoft.com",
	"pre.footprintpredict.com",
	"statsfe2.update.microsoft.com.akadns.net",
}

// hostsFileMarkerStart and hostsFileMarkerEnd delimit the CleanForge section in the hosts file.
const (
	hostsFileMarkerStart = "# --- CleanForge Telemetry Block Start ---"
	hostsFileMarkerEnd   = "# --- CleanForge Telemetry Block End ---"
)

// allTweaks defines every available registry-based privacy tweak.
var allTweaks = []registryTweak{
	{
		id:          "disable_telemetry",
		name:        "Disable Telemetry",
		description: "Disables Windows diagnostic data collection (AllowTelemetry=0)",
		category:    "telemetry",
		entries: []registryEntry{
			{registry.LOCAL_MACHINE, `SOFTWARE\Policies\Microsoft\Windows\DataCollection`, "AllowTelemetry", 0},
		},
	},
	{
		id:          "disable_activity_history",
		name:        "Disable Activity History",
		description: "Prevents Windows from tracking and sending your activity history",
		category:    "tracking",
		entries: []registryEntry{
			{registry.LOCAL_MACHINE, `SOFTWARE\Policies\Microsoft\Windows\System`, "EnableActivityFeed", 0},
			{registry.LOCAL_MACHINE, `SOFTWARE\Policies\Microsoft\Windows\System`, "PublishUserActivities", 0},
		},
	},
	{
		id:          "disable_location",
		name:        "Disable Location Tracking",
		description: "Denies app access to your device location",
		category:    "tracking",
		entries: []registryEntry{
			// This one is a string value, handled specially
		},
	},
	{
		id:          "disable_advertising_id",
		name:        "Disable Advertising ID",
		description: "Prevents apps from using your advertising ID for targeted ads",
		category:    "ads",
		entries: []registryEntry{
			{registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows\CurrentVersion\AdvertisingInfo`, "Enabled", 0},
		},
	},
	{
		id:          "disable_cortana",
		name:        "Disable Cortana",
		description: "Disables Cortana assistant and its data collection",
		category:    "cortana",
		entries: []registryEntry{
			{registry.LOCAL_MACHINE, `SOFTWARE\Policies\Microsoft\Windows\Windows Search`, "AllowCortana", 0},
		},
	},
	{
		id:          "disable_bing_search",
		name:        "Disable Bing Search in Start Menu",
		description: "Removes Bing web search suggestions from the Start Menu search",
		category:    "cortana",
		entries: []registryEntry{
			{registry.CURRENT_USER, `SOFTWARE\Policies\Microsoft\Windows\Explorer`, "DisableSearchBoxSuggestions", 1},
		},
	},
	{
		id:          "disable_feedback",
		name:        "Disable Feedback Requests",
		description: "Stops Windows from asking for feedback",
		category:    "telemetry",
		entries: []registryEntry{
			{registry.CURRENT_USER, `SOFTWARE\Microsoft\Siuf\Rules`, "NumberOfSIUFInPeriod", 0},
		},
	},
	{
		id:          "disable_tailored_experiences",
		name:        "Disable Tailored Experiences",
		description: "Prevents Microsoft from using diagnostic data for personalized tips and ads",
		category:    "ads",
		entries: []registryEntry{
			{registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows\CurrentVersion\Privacy`, "TailoredExperiencesWithDiagnosticDataEnabled", 0},
		},
	},
	{
		id:          "disable_tips",
		name:        "Disable Tips and Suggestions",
		description: "Disables Windows tips, suggestions, and recommended content",
		category:    "ads",
		entries: []registryEntry{
			{registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows\CurrentVersion\ContentDeliveryManager`, "SubscribedContent-338389Enabled", 0},
			{registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows\CurrentVersion\ContentDeliveryManager`, "SoftLandingEnabled", 0},
			{registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows\CurrentVersion\ContentDeliveryManager`, "SystemPaneSuggestionsEnabled", 0},
		},
	},
	{
		id:          "block_telemetry_hosts",
		name:        "Block Telemetry Hosts",
		description: "Adds telemetry domains to the hosts file to block data collection at the network level",
		category:    "telemetry",
		entries:     nil, // Handled specially via hosts file
	},
	{
		id:          "disable_wifi_sense",
		name:        "Disable Wi-Fi Sense",
		description: "Prevents automatic connection to suggested open hotspots and shared networks",
		category:    "tracking",
		entries: []registryEntry{
			{registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\WcmSvc\wifinetworkmanager\config`, "AutoConnectAllowedOEM", 0},
		},
	},
	{
		id:          "disable_error_reporting",
		name:        "Disable Windows Error Reporting",
		description: "Stops Windows from sending error reports to Microsoft",
		category:    "telemetry",
		entries: []registryEntry{
			{registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\Windows Error Reporting`, "Disabled", 1},
		},
	},
}

// GetPrivacyTweaks returns all available privacy tweaks with their current applied state.
func GetPrivacyTweaks() ([]PrivacyTweak, error) {
	var tweaks []PrivacyTweak

	for _, t := range allTweaks {
		tweak := PrivacyTweak{
			ID:          t.id,
			Name:        t.name,
			Description: t.description,
			Category:    t.category,
		}

		applied := isTweakApplied(t)
		tweak.Applied = applied
		tweak.Enabled = applied

		tweaks = append(tweaks, tweak)
	}

	return tweaks, nil
}

// ApplyTweak applies a single privacy tweak by its ID.
func ApplyTweak(tweakID string) error {
	for _, t := range allTweaks {
		if t.id == tweakID {
			return applyTweak(t)
		}
	}
	return fmt.Errorf("unknown tweak ID: %s", tweakID)
}

// ApplyAll applies all available privacy tweaks.
func ApplyAll() error {
	var lastErr error
	for _, t := range allTweaks {
		if err := applyTweak(t); err != nil {
			lastErr = fmt.Errorf("failed to apply %s: %w", t.id, err)
		}
	}
	return lastErr
}

// RestoreAll restores all privacy tweaks to Windows defaults.
func RestoreAll() error {
	var lastErr error
	for _, t := range allTweaks {
		if err := restoreTweak(t); err != nil {
			lastErr = fmt.Errorf("failed to restore %s: %w", t.id, err)
		}
	}
	return lastErr
}

// applyTweak applies a single tweak's registry changes or hosts file modifications.
func applyTweak(t registryTweak) error {
	// Special case: location tracking uses a string registry value
	if t.id == "disable_location" {
		return applyLocationTweak()
	}

	// Special case: hosts file blocking
	if t.id == "block_telemetry_hosts" {
		return applyHostsBlock()
	}

	// Apply all registry entries for this tweak
	for _, entry := range t.entries {
		if err := setRegistryDWORD(entry.rootKey, entry.path, entry.name, entry.value); err != nil {
			return fmt.Errorf("failed to set %s\\%s: %w", entry.path, entry.name, err)
		}
	}

	return nil
}

// restoreTweak restores a single tweak to Windows defaults.
func restoreTweak(t registryTweak) error {
	// Special case: location tracking
	if t.id == "disable_location" {
		return restoreLocationTweak()
	}

	// Special case: hosts file
	if t.id == "block_telemetry_hosts" {
		return removeHostsBlock()
	}

	// Delete all registry values for this tweak to restore defaults
	for _, entry := range t.entries {
		if err := deleteRegistryValue(entry.rootKey, entry.path, entry.name); err != nil {
			// Ignore errors from values that don't exist
			continue
		}
	}

	return nil
}

// isTweakApplied checks if a tweak is currently applied by reading its registry
// values or checking the hosts file.
func isTweakApplied(t registryTweak) bool {
	// Special case: location tracking
	if t.id == "disable_location" {
		return isLocationTweakApplied()
	}

	// Special case: hosts file
	if t.id == "block_telemetry_hosts" {
		return isHostsBlockApplied()
	}

	// Check all registry entries
	for _, entry := range t.entries {
		val, err := readRegistryDWORD(entry.rootKey, entry.path, entry.name)
		if err != nil {
			return false
		}
		if val != entry.value {
			return false
		}
	}

	return true
}

// --- Location Tweak (string value) ---

func applyLocationTweak() error {
	keyPath := `SOFTWARE\Microsoft\Windows\CurrentVersion\CapabilityAccessManager\ConsentStore\location`
	k, _, err := registry.CreateKey(registry.CURRENT_USER, keyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to create location key: %w", err)
	}
	defer k.Close()

	if err := k.SetStringValue("Value", "Deny"); err != nil {
		return fmt.Errorf("failed to set location Value: %w", err)
	}

	return nil
}

func restoreLocationTweak() error {
	keyPath := `SOFTWARE\Microsoft\Windows\CurrentVersion\CapabilityAccessManager\ConsentStore\location`
	k, err := registry.OpenKey(registry.CURRENT_USER, keyPath, registry.SET_VALUE)
	if err != nil {
		return nil // Key doesn't exist, nothing to restore
	}
	defer k.Close()

	if err := k.SetStringValue("Value", "Allow"); err != nil {
		return fmt.Errorf("failed to restore location Value: %w", err)
	}

	return nil
}

func isLocationTweakApplied() bool {
	keyPath := `SOFTWARE\Microsoft\Windows\CurrentVersion\CapabilityAccessManager\ConsentStore\location`
	k, err := registry.OpenKey(registry.CURRENT_USER, keyPath, registry.READ)
	if err != nil {
		return false
	}
	defer k.Close()

	val, _, err := k.GetStringValue("Value")
	if err != nil {
		return false
	}

	return strings.EqualFold(val, "Deny")
}

// --- Hosts File Management ---

func getHostsFilePath() string {
	systemRoot := os.Getenv("SystemRoot")
	if systemRoot == "" {
		systemRoot = `C:\Windows`
	}
	return filepath.Join(systemRoot, "System32", "drivers", "etc", "hosts")
}

func applyHostsBlock() error {
	hostsPath := getHostsFilePath()

	// First, remove any existing CleanForge block to avoid duplicates
	if err := removeHostsBlock(); err != nil {
		// Non-fatal; continue with appending
	}

	// Read current hosts file content
	content, err := os.ReadFile(hostsPath)
	if err != nil {
		return fmt.Errorf("failed to read hosts file: %w", err)
	}

	// Build the block to append
	var block strings.Builder
	block.WriteString("\n")
	block.WriteString(hostsFileMarkerStart)
	block.WriteString("\n")
	for _, host := range telemetryHosts {
		block.WriteString(fmt.Sprintf("0.0.0.0 %s\n", host))
	}
	block.WriteString(hostsFileMarkerEnd)
	block.WriteString("\n")

	// Append block to hosts file
	newContent := string(content) + block.String()
	if err := os.WriteFile(hostsPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write hosts file: %w", err)
	}

	return nil
}

func removeHostsBlock() error {
	hostsPath := getHostsFilePath()

	file, err := os.Open(hostsPath)
	if err != nil {
		return fmt.Errorf("failed to open hosts file: %w", err)
	}
	defer file.Close()

	var newLines []string
	inBlock := false
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		if strings.TrimSpace(line) == hostsFileMarkerStart {
			inBlock = true
			continue
		}
		if strings.TrimSpace(line) == hostsFileMarkerEnd {
			inBlock = false
			continue
		}

		if !inBlock {
			newLines = append(newLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read hosts file: %w", err)
	}

	// Write the cleaned content back
	output := strings.Join(newLines, "\n") + "\n"
	if err := os.WriteFile(hostsPath, []byte(output), 0644); err != nil {
		return fmt.Errorf("failed to write hosts file: %w", err)
	}

	return nil
}

func isHostsBlockApplied() bool {
	hostsPath := getHostsFilePath()

	file, err := os.Open(hostsPath)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == hostsFileMarkerStart {
			return true
		}
	}

	return false
}

// --- Registry Helpers ---

// setRegistryDWORD creates or opens the specified key and sets a DWORD value.
func setRegistryDWORD(rootKey registry.Key, path, name string, value uint32) error {
	k, _, err := registry.CreateKey(rootKey, path, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to create/open key %s: %w", path, err)
	}
	defer k.Close()

	if err := k.SetDWordValue(name, value); err != nil {
		return fmt.Errorf("failed to set %s: %w", name, err)
	}

	return nil
}

// readRegistryDWORD reads a DWORD value from the registry.
func readRegistryDWORD(rootKey registry.Key, path, name string) (uint32, error) {
	k, err := registry.OpenKey(rootKey, path, registry.READ)
	if err != nil {
		return 0, err
	}
	defer k.Close()

	val, _, err := k.GetIntegerValue(name)
	if err != nil {
		return 0, err
	}

	return uint32(val), nil
}

// deleteRegistryValue deletes a named value from the given registry key.
func deleteRegistryValue(rootKey registry.Key, path, name string) error {
	k, err := registry.OpenKey(rootKey, path, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	return k.DeleteValue(name)
}
