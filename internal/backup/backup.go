package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/windows/registry"
)

// BackupState holds the complete system state snapshot used for backup and restore.
type BackupState struct {
	Timestamp    string                       `json:"timestamp"`
	RegistryKeys map[string]RegistryBackup    `json:"registryKeys"`
	Services     map[string]string            `json:"services"`  // service name -> original start type
	PowerPlan    string                       `json:"powerPlan"` // original active power plan GUID
}

// RegistryBackup holds a single registry value's backup information.
type RegistryBackup struct {
	Path      string      `json:"path"`
	ValueName string      `json:"valueName"`
	Value     interface{} `json:"value"`
	Type      string      `json:"type"`    // "string", "dword", "qword", "none"
	Existed   bool        `json:"existed"` // if the key/value existed before we changed it
}

// backupFilename is the name of the backup state file.
const backupFilename = "backup_state.json"

// state holds the current in-memory backup state.
var state *BackupState

// init initializes the in-memory state.
func init() {
	state = newEmptyState()
}

// newEmptyState creates a fresh empty BackupState.
func newEmptyState() *BackupState {
	return &BackupState{
		Timestamp:    time.Now().Format(time.RFC3339),
		RegistryKeys: make(map[string]RegistryBackup),
		Services:     make(map[string]string),
		PowerPlan:    "",
	}
}

// GetBackupPath returns the directory where backup files are stored.
// Creates the directory if it does not exist.
func GetBackupPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to USERPROFILE environment variable
		homeDir = os.Getenv("USERPROFILE")
		if homeDir == "" {
			homeDir = "C:\\Users\\Default"
		}
	}
	backupDir := filepath.Join(homeDir, ".cleanforge", "backups")
	// Ensure directory exists
	_ = os.MkdirAll(backupDir, 0755)
	return backupDir
}

// getBackupFilePath returns the full path to the backup state JSON file.
func getBackupFilePath() string {
	return filepath.Join(GetBackupPath(), backupFilename)
}

// SaveRegistryValue reads the current value of a registry key and saves it to the
// in-memory backup state. rootKey should be one of: "HKLM", "HKCU", "HKCR", "HKU", "HKCC".
// path is the registry subkey path, and valueName is the specific value name to back up.
func SaveRegistryValue(rootKey, path, valueName string) error {
	root, err := parseRootKey(rootKey)
	if err != nil {
		return err
	}

	// Construct a unique key for the map
	mapKey := fmt.Sprintf("%s\\%s\\%s", rootKey, path, valueName)

	// Try to open the registry key
	key, err := registry.OpenKey(root, path, registry.QUERY_VALUE)
	if err != nil {
		// Key doesn't exist; record that it didn't exist
		state.RegistryKeys[mapKey] = RegistryBackup{
			Path:      fmt.Sprintf("%s\\%s", rootKey, path),
			ValueName: valueName,
			Value:     nil,
			Type:      "none",
			Existed:   false,
		}
		return nil
	}
	defer key.Close()

	// Read the value
	val, valType, err := key.GetValue(valueName, nil)
	if err != nil {
		// Value doesn't exist under the key
		state.RegistryKeys[mapKey] = RegistryBackup{
			Path:      fmt.Sprintf("%s\\%s", rootKey, path),
			ValueName: valueName,
			Value:     nil,
			Type:      "none",
			Existed:   false,
		}
		return nil
	}

	// Determine the type and read the actual value
	backup := RegistryBackup{
		Path:      fmt.Sprintf("%s\\%s", rootKey, path),
		ValueName: valueName,
		Existed:   true,
	}

	switch valType {
	case registry.SZ, registry.EXPAND_SZ:
		strVal, _, err := key.GetStringValue(valueName)
		if err == nil {
			backup.Value = strVal
			backup.Type = "string"
		}
	case registry.DWORD:
		dwordVal, _, err := key.GetIntegerValue(valueName)
		if err == nil {
			backup.Value = uint32(dwordVal)
			backup.Type = "dword"
		}
	case registry.QWORD:
		qwordVal, _, err := key.GetIntegerValue(valueName)
		if err == nil {
			backup.Value = qwordVal
			backup.Type = "qword"
		}
	default:
		// Store raw bytes for unsupported types
		backup.Value = val
		backup.Type = "binary"
	}

	state.RegistryKeys[mapKey] = backup
	return nil
}

// SaveServiceState reads the current start type of a Windows service and saves it
// to the in-memory backup state.
func SaveServiceState(serviceName string) error {
	out, err := exec.Command("sc", "qc", serviceName).Output()
	if err != nil {
		return fmt.Errorf("failed to query service '%s': %w", serviceName, err)
	}

	startType := parseServiceStartType(string(out))
	if startType == "" {
		return fmt.Errorf("could not parse start type for service '%s'", serviceName)
	}

	state.Services[serviceName] = startType
	return nil
}

// SavePowerPlan reads the currently active power plan GUID and saves it to the
// in-memory backup state.
func SavePowerPlan() error {
	out, err := exec.Command("powercfg", "/getactivescheme").Output()
	if err != nil {
		return fmt.Errorf("failed to get active power plan: %w", err)
	}

	guid := parsePowerPlanGUID(string(out))
	if guid == "" {
		return fmt.Errorf("could not parse power plan GUID from output: %s", string(out))
	}

	state.PowerPlan = guid
	return nil
}

// Save writes the current in-memory backup state to the JSON file on disk.
// Uses MarshalIndent for human-readable output.
func Save() error {
	state.Timestamp = time.Now().Format(time.RFC3339)

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal backup state: %w", err)
	}

	filePath := getBackupFilePath()

	// Ensure the directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}

	return nil
}

// Load reads the backup state from the JSON file on disk and returns it.
// Also loads it into the in-memory state for subsequent operations.
func Load() (*BackupState, error) {
	filePath := getBackupFilePath()

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup file: %w", err)
	}

	loaded := &BackupState{}
	if err := json.Unmarshal(data, loaded); err != nil {
		return nil, fmt.Errorf("failed to parse backup file: %w", err)
	}

	// Ensure maps are initialized
	if loaded.RegistryKeys == nil {
		loaded.RegistryKeys = make(map[string]RegistryBackup)
	}
	if loaded.Services == nil {
		loaded.Services = make(map[string]string)
	}

	state = loaded
	return loaded, nil
}

// RestoreRegistry reads the backup state and restores all registry values.
// If a value did not exist before (Existed=false), the value is deleted.
// If it did exist, it is set back to its original value.
func RestoreRegistry() error {
	if state == nil || len(state.RegistryKeys) == 0 {
		return nil
	}

	var errors []string

	for _, backup := range state.RegistryKeys {
		err := restoreSingleRegistryValue(backup)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s\\%s: %v", backup.Path, backup.ValueName, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("some registry values could not be restored:\n%s", strings.Join(errors, "\n"))
	}

	return nil
}

// restoreSingleRegistryValue restores a single registry value from its backup.
func restoreSingleRegistryValue(backup RegistryBackup) error {
	// Parse the root key and subpath from the full path
	rootKeyStr, subPath, err := splitRegistryPath(backup.Path)
	if err != nil {
		return err
	}

	root, err := parseRootKey(rootKeyStr)
	if err != nil {
		return err
	}

	if !backup.Existed {
		// The value didn't exist before; delete it
		key, err := registry.OpenKey(root, subPath, registry.SET_VALUE)
		if err != nil {
			// Key doesn't exist anyway, nothing to delete
			return nil
		}
		defer key.Close()

		err = key.DeleteValue(backup.ValueName)
		if err != nil {
			// Value might not exist, which is fine
			return nil
		}
		return nil
	}

	// The value existed; restore it
	key, err := registry.OpenKey(root, subPath, registry.SET_VALUE)
	if err != nil {
		// Try to create the key
		key, _, err = registry.CreateKey(root, subPath, registry.SET_VALUE)
		if err != nil {
			return fmt.Errorf("failed to open/create registry key: %w", err)
		}
	}
	defer key.Close()

	switch backup.Type {
	case "string":
		strVal, ok := backup.Value.(string)
		if !ok {
			return fmt.Errorf("expected string value, got %T", backup.Value)
		}
		return key.SetStringValue(backup.ValueName, strVal)

	case "dword":
		var dwordVal uint32
		switch v := backup.Value.(type) {
		case float64:
			// JSON unmarshals numbers as float64
			dwordVal = uint32(v)
		case uint32:
			dwordVal = v
		case int:
			dwordVal = uint32(v)
		default:
			return fmt.Errorf("expected numeric value for DWORD, got %T", backup.Value)
		}
		return key.SetDWordValue(backup.ValueName, dwordVal)

	case "qword":
		var qwordVal uint64
		switch v := backup.Value.(type) {
		case float64:
			qwordVal = uint64(v)
		case uint64:
			qwordVal = v
		case int:
			qwordVal = uint64(v)
		default:
			return fmt.Errorf("expected numeric value for QWORD, got %T", backup.Value)
		}
		return key.SetQWordValue(backup.ValueName, qwordVal)

	case "none":
		// Nothing to restore
		return nil

	default:
		return fmt.Errorf("unsupported registry value type: %s", backup.Type)
	}
}

// RestoreServices restores all service start types from the backup state.
func RestoreServices() error {
	if state == nil || len(state.Services) == 0 {
		return nil
	}

	var errors []string

	for serviceName, startType := range state.Services {
		scStartType := mapStartTypeToSC(startType)
		if scStartType == "" {
			errors = append(errors, fmt.Sprintf("%s: unknown start type '%s'", serviceName, startType))
			continue
		}

		cmd := exec.Command("sc", "config", serviceName, "start=", scStartType)
		if out, err := cmd.CombinedOutput(); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v (%s)", serviceName, err, strings.TrimSpace(string(out))))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("some services could not be restored:\n%s", strings.Join(errors, "\n"))
	}

	return nil
}

// RestorePowerPlan restores the original active power plan from the backup state.
func RestorePowerPlan() error {
	if state == nil || state.PowerPlan == "" {
		return nil
	}

	cmd := exec.Command("powercfg", "/setactive", state.PowerPlan)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to restore power plan %s: %v (%s)", state.PowerPlan, err, strings.TrimSpace(string(out)))
	}

	return nil
}

// RestoreAll performs a full restoration: registry values, services, and power plan.
func RestoreAll() error {
	var errors []string

	if err := RestoreRegistry(); err != nil {
		errors = append(errors, fmt.Sprintf("Registry: %v", err))
	}

	if err := RestoreServices(); err != nil {
		errors = append(errors, fmt.Sprintf("Services: %v", err))
	}

	if err := RestorePowerPlan(); err != nil {
		errors = append(errors, fmt.Sprintf("PowerPlan: %v", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("restore completed with errors:\n%s", strings.Join(errors, "\n"))
	}

	return nil
}

// HasBackup checks whether a backup file exists on disk.
func HasBackup() bool {
	filePath := getBackupFilePath()
	info, err := os.Stat(filePath)
	if err != nil {
		return false
	}
	return !info.IsDir() && info.Size() > 0
}

// parseRootKey converts a root key string to a registry.Key constant.
func parseRootKey(rootKey string) (registry.Key, error) {
	switch strings.ToUpper(rootKey) {
	case "HKLM", "HKEY_LOCAL_MACHINE":
		return registry.LOCAL_MACHINE, nil
	case "HKCU", "HKEY_CURRENT_USER":
		return registry.CURRENT_USER, nil
	case "HKCR", "HKEY_CLASSES_ROOT":
		return registry.CLASSES_ROOT, nil
	case "HKU", "HKEY_USERS":
		return registry.USERS, nil
	case "HKCC", "HKEY_CURRENT_CONFIG":
		return registry.CURRENT_CONFIG, nil
	default:
		return 0, fmt.Errorf("unknown registry root key: %s", rootKey)
	}
}

// splitRegistryPath splits a full registry path like "HKLM\SOFTWARE\Test" into
// the root key string ("HKLM") and the subpath ("SOFTWARE\Test").
func splitRegistryPath(fullPath string) (string, string, error) {
	// Normalize path separators
	normalized := strings.ReplaceAll(fullPath, "/", "\\")

	idx := strings.Index(normalized, "\\")
	if idx < 0 {
		return "", "", fmt.Errorf("invalid registry path (no separator): %s", fullPath)
	}

	rootStr := normalized[:idx]
	subPath := normalized[idx+1:]

	return rootStr, subPath, nil
}

// parseServiceStartType extracts the start type from the output of `sc qc <service>`.
// The output contains a line like: "START_TYPE : 2  AUTO_START"
// We extract the descriptive name (AUTO_START, DEMAND_START, DISABLED, etc.)
func parseServiceStartType(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "START_TYPE") {
			// Format: "START_TYPE         : 2   AUTO_START"
			// or:     "START_TYPE         : 3   DEMAND_START"
			// or:     "START_TYPE         : 4   DISABLED"
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) < 2 {
				continue
			}
			value := strings.TrimSpace(parts[1])

			// The value contains both the numeric code and the name
			// Extract the name (last token)
			tokens := strings.Fields(value)
			if len(tokens) >= 2 {
				return tokens[len(tokens)-1]
			}
			if len(tokens) == 1 {
				return tokens[0]
			}
		}
	}
	return ""
}

// mapStartTypeToSC maps a service start type name back to the sc config parameter.
func mapStartTypeToSC(startType string) string {
	switch strings.ToUpper(startType) {
	case "AUTO_START":
		return "auto"
	case "DEMAND_START":
		return "demand"
	case "DISABLED":
		return "disabled"
	case "BOOT_START":
		return "boot"
	case "SYSTEM_START":
		return "system"
	case "AUTO":
		return "auto"
	case "DEMAND":
		return "demand"
	default:
		// Try to return as-is if it matches an sc parameter
		lower := strings.ToLower(startType)
		switch lower {
		case "auto", "demand", "disabled", "boot", "system":
			return lower
		}
		return ""
	}
}

// parsePowerPlanGUID extracts the power plan GUID from the output of `powercfg /getactivescheme`.
// Example output: "Power Scheme GUID: 381b4222-f694-41f0-9685-ff5bb260df2e  (Balanced)"
func parsePowerPlanGUID(output string) string {
	// Look for the GUID pattern after "GUID:"
	idx := strings.Index(output, "GUID:")
	if idx < 0 {
		// Try without colon
		idx = strings.Index(output, "GUID")
		if idx < 0 {
			return ""
		}
		idx += 4
	} else {
		idx += 5 // len("GUID:")
	}

	remaining := strings.TrimSpace(output[idx:])

	// The GUID is the next whitespace-delimited token
	// Format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	fields := strings.Fields(remaining)
	if len(fields) == 0 {
		return ""
	}

	guid := fields[0]
	// Validate it looks like a GUID (contains dashes and is ~36 chars)
	if len(guid) >= 36 && strings.Count(guid, "-") == 4 {
		return guid
	}

	return ""
}
