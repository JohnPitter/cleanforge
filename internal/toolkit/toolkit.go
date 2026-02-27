package toolkit

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

// ToolResult holds the outcome of a system tool operation.
type ToolResult struct {
	Name    string   `json:"name"`
	Success bool     `json:"success"`
	Output  string   `json:"output"`
	Errors  []string `json:"errors"`
}

// BloatwareApp represents a Windows Store app that may be considered bloatware.
type BloatwareApp struct {
	Name        string `json:"name"`
	PackageName string `json:"packageName"`
	Publisher   string `json:"publisher"`
	Installed   bool   `json:"installed"`
}

// knownBloatware is the list of package names considered bloatware.
var knownBloatware = []string{
	"Microsoft.BingNews",
	"Microsoft.BingWeather",
	"Microsoft.GetHelp",
	"Microsoft.Getstarted",
	"Microsoft.MicrosoftOfficeHub",
	"Microsoft.MicrosoftSolitaireCollection",
	"Microsoft.People",
	"Microsoft.WindowsFeedbackHub",
	"Microsoft.Xbox.TCUI",
	"Microsoft.XboxGameOverlay",
	"Microsoft.XboxGamingOverlay",
	"Microsoft.XboxIdentityProvider",
	"Microsoft.XboxSpeechToTextOverlay",
	"Microsoft.YourPhone",
	"Microsoft.ZuneMusic",
	"Microsoft.ZuneVideo",
	"Clipchamp.Clipchamp",
	"Microsoft.Todos",
	"Microsoft.PowerAutomateDesktop",
	"Microsoft.549981C3F5F10", // Cortana
	"Disney.37853FC22B2CE",
	"SpotifyAB.SpotifyMusic",
	"king.com.CandyCrushSaga",
	"BytedancePte.Ltd.TikTok",
}

// IsAdmin checks whether the current process is running with administrator privileges.
func IsAdmin() bool {
	var sid *windows.SID

	// Create a SID for the BUILTIN\Administrators group.
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid,
	)
	if err != nil {
		return false
	}
	defer windows.FreeSid(sid)

	// CheckTokenMembership requires an impersonation token; passing 0 uses the
	// process token directly which works when the process itself is elevated.
	token := windows.Token(0)
	isMember, err := token.IsMember(sid)
	if err != nil {
		return false
	}

	return isMember
}

// RunSFC executes the System File Checker (sfc /scannow) and returns the result.
// Requires administrator privileges.
func RunSFC() (*ToolResult, error) {
	result := &ToolResult{
		Name: "System File Checker (SFC)",
	}

	if !IsAdmin() {
		result.Success = false
		result.Output = "This tool requires administrator privileges. Please run CleanForge as administrator."
		result.Errors = append(result.Errors, "not running as administrator")
		return result, nil
	}

	out, err := exec.Command("sfc", "/scannow").CombinedOutput()
	output := strings.TrimSpace(string(out))
	result.Output = output

	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("sfc exited with error: %s", err.Error()))
		return result, nil
	}

	result.Success = true
	return result, nil
}

// RunDISM executes DISM to repair the Windows component store.
// Requires administrator privileges.
func RunDISM() (*ToolResult, error) {
	result := &ToolResult{
		Name: "DISM (Deployment Image Servicing)",
	}

	if !IsAdmin() {
		result.Success = false
		result.Output = "This tool requires administrator privileges. Please run CleanForge as administrator."
		result.Errors = append(result.Errors, "not running as administrator")
		return result, nil
	}

	out, err := exec.Command("DISM", "/Online", "/Cleanup-Image", "/RestoreHealth").CombinedOutput()
	output := strings.TrimSpace(string(out))
	result.Output = output

	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("DISM exited with error: %s", err.Error()))
		return result, nil
	}

	result.Success = true
	return result, nil
}

// GetBloatwareApps queries installed AppX packages and returns those that match
// the known bloatware list, along with their installation status.
func GetBloatwareApps() ([]BloatwareApp, error) {
	// Query all installed AppX packages via PowerShell
	psCmd := `Get-AppxPackage | Select-Object Name, PackageFullName, Publisher | ConvertTo-Csv -NoTypeInformation`
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to query AppX packages: %s - %w", strings.TrimSpace(string(out)), err)
	}

	// Parse CSV output to build a map of installed package names
	installedPackages := make(map[string]struct {
		fullName  string
		publisher string
	})

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for i, line := range lines {
		if i == 0 {
			// Skip CSV header
			continue
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := parseCSVLine(line)
		if len(fields) >= 3 {
			name := strings.Trim(fields[0], `"`)
			fullName := strings.Trim(fields[1], `"`)
			publisher := strings.Trim(fields[2], `"`)
			installedPackages[name] = struct {
				fullName  string
				publisher string
			}{fullName, publisher}
		}
	}

	// Build the result list
	var apps []BloatwareApp
	for _, pkgName := range knownBloatware {
		app := BloatwareApp{
			Name:        friendlyName(pkgName),
			PackageName: pkgName,
			Installed:   false,
		}

		if info, found := installedPackages[pkgName]; found {
			app.Installed = true
			app.Publisher = info.publisher
			// Use the full package name for removal
			if info.fullName != "" {
				app.PackageName = pkgName
			}
		}

		apps = append(apps, app)
	}

	return apps, nil
}

// RemoveBloatware removes the specified AppX packages.
func RemoveBloatware(packageNames []string) (*ToolResult, error) {
	result := &ToolResult{
		Name: "Remove Bloatware",
	}

	if len(packageNames) == 0 {
		result.Success = true
		result.Output = "No packages specified for removal."
		return result, nil
	}

	var outputs []string
	var errors []string

	for _, pkgName := range packageNames {
		psCmd := fmt.Sprintf(`Get-AppxPackage -Name "%s" | Remove-AppxPackage`, pkgName)
		out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).CombinedOutput()
		output := strings.TrimSpace(string(out))

		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to remove %s: %s - %s", pkgName, err.Error(), output))
		} else {
			outputs = append(outputs, fmt.Sprintf("Removed: %s", pkgName))
		}
	}

	result.Output = strings.Join(outputs, "\n")
	result.Errors = errors
	result.Success = len(errors) == 0

	if len(outputs) > 0 && len(errors) > 0 {
		result.Output += "\n\nSome packages failed to remove."
	}

	return result, nil
}

// RepairWindowsUpdate stops the Windows Update service, clears the
// SoftwareDistribution folder, and restarts the service.
func RepairWindowsUpdate() (*ToolResult, error) {
	result := &ToolResult{
		Name: "Repair Windows Update",
	}

	if !IsAdmin() {
		result.Success = false
		result.Output = "This tool requires administrator privileges. Please run CleanForge as administrator."
		result.Errors = append(result.Errors, "not running as administrator")
		return result, nil
	}

	var outputs []string
	var errors []string

	// Stop the Windows Update service
	out, err := exec.Command("net", "stop", "wuauserv").CombinedOutput()
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to stop wuauserv: %s - %s", err.Error(), strings.TrimSpace(string(out))))
	} else {
		outputs = append(outputs, "Stopped Windows Update service")
	}

	// Stop BITS service
	out, err = exec.Command("net", "stop", "bits").CombinedOutput()
	if err != nil {
		// BITS might not be running; non-critical
		outputs = append(outputs, fmt.Sprintf("BITS service: %s", strings.TrimSpace(string(out))))
	} else {
		outputs = append(outputs, "Stopped BITS service")
	}

	// Delete SoftwareDistribution folder contents
	sdPath := filepath.Join(os.Getenv("SystemRoot"), "SoftwareDistribution")
	if _, err := os.Stat(sdPath); err == nil {
		err = os.RemoveAll(sdPath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to delete SoftwareDistribution: %s", err.Error()))
		} else {
			outputs = append(outputs, "Deleted SoftwareDistribution folder")
		}
	}

	// Restart the Windows Update service
	out, err = exec.Command("net", "start", "wuauserv").CombinedOutput()
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to start wuauserv: %s - %s", err.Error(), strings.TrimSpace(string(out))))
	} else {
		outputs = append(outputs, "Restarted Windows Update service")
	}

	// Restart BITS service
	out, err = exec.Command("net", "start", "bits").CombinedOutput()
	if err != nil {
		outputs = append(outputs, fmt.Sprintf("BITS restart: %s", strings.TrimSpace(string(out))))
	} else {
		outputs = append(outputs, "Restarted BITS service")
	}

	result.Output = strings.Join(outputs, "\n")
	result.Errors = errors
	result.Success = len(errors) == 0

	return result, nil
}

// RebuildIconCache deletes the Windows icon cache files and restarts Explorer
// to force a rebuild.
func RebuildIconCache() (*ToolResult, error) {
	result := &ToolResult{
		Name: "Rebuild Icon Cache",
	}

	var outputs []string
	var errors []string

	// Icon cache location
	localAppData := os.Getenv("LOCALAPPDATA")
	cacheDir := filepath.Join(localAppData, "Microsoft", "Windows", "Explorer")

	// Kill explorer.exe first to release file handles
	out, err := exec.Command("taskkill", "/f", "/im", "explorer.exe").CombinedOutput()
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to kill explorer.exe: %s - %s", err.Error(), strings.TrimSpace(string(out))))
	} else {
		outputs = append(outputs, "Stopped Explorer")
	}

	// Delete icon cache files
	pattern := filepath.Join(cacheDir, "iconcache*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to glob icon cache files: %s", err.Error()))
	} else {
		deletedCount := 0
		for _, match := range matches {
			if err := os.Remove(match); err != nil {
				errors = append(errors, fmt.Sprintf("Failed to delete %s: %s", filepath.Base(match), err.Error()))
			} else {
				deletedCount++
			}
		}
		outputs = append(outputs, fmt.Sprintf("Deleted %d icon cache files", deletedCount))
	}

	// Also try thumbcache files
	thumbPattern := filepath.Join(cacheDir, "thumbcache*")
	thumbMatches, _ := filepath.Glob(thumbPattern)
	for _, match := range thumbMatches {
		_ = os.Remove(match)
	}

	// Restart explorer.exe
	err = exec.Command("cmd", "/c", "start", "explorer.exe").Start()
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to restart explorer.exe: %s", err.Error()))
	} else {
		outputs = append(outputs, "Restarted Explorer")
	}

	result.Output = strings.Join(outputs, "\n")
	result.Errors = errors
	result.Success = len(errors) == 0

	return result, nil
}

// RebuildFontCache stops the Font Cache service, deletes cached font data,
// and restarts the service.
func RebuildFontCache() (*ToolResult, error) {
	result := &ToolResult{
		Name: "Rebuild Font Cache",
	}

	if !IsAdmin() {
		result.Success = false
		result.Output = "This tool requires administrator privileges. Please run CleanForge as administrator."
		result.Errors = append(result.Errors, "not running as administrator")
		return result, nil
	}

	var outputs []string
	var errors []string

	// Stop Font Cache service
	out, err := exec.Command("net", "stop", "FontCache").CombinedOutput()
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to stop FontCache: %s - %s", err.Error(), strings.TrimSpace(string(out))))
	} else {
		outputs = append(outputs, "Stopped Font Cache service")
	}

	// Also stop the Font Cache 3.0.0.0 service
	out, err = exec.Command("net", "stop", "FontCache3.0.0.0").CombinedOutput()
	if err != nil {
		// This service might not exist on all systems; non-critical
		outputs = append(outputs, "FontCache 3.0.0.0 service not found or already stopped")
	} else {
		outputs = append(outputs, "Stopped Font Cache 3.0.0.0 service")
	}

	// Delete font cache files
	systemRoot := os.Getenv("SystemRoot")
	fontCachePath := filepath.Join(systemRoot, "ServiceProfiles", "LocalService", "AppData", "Local", "FontCache")

	if _, err := os.Stat(fontCachePath); err == nil {
		entries, err := os.ReadDir(fontCachePath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to read font cache directory: %s", err.Error()))
		} else {
			deletedCount := 0
			for _, entry := range entries {
				if strings.HasSuffix(strings.ToLower(entry.Name()), ".dat") || strings.HasPrefix(strings.ToLower(entry.Name()), "fontcache") {
					fullPath := filepath.Join(fontCachePath, entry.Name())
					if err := os.Remove(fullPath); err != nil {
						errors = append(errors, fmt.Sprintf("Failed to delete %s: %s", entry.Name(), err.Error()))
					} else {
						deletedCount++
					}
				}
			}
			outputs = append(outputs, fmt.Sprintf("Deleted %d font cache files", deletedCount))
		}
	}

	// Also delete the FNTCACHE.DAT in System32
	fntCachePath := filepath.Join(systemRoot, "System32", "FNTCACHE.DAT")
	if _, err := os.Stat(fntCachePath); err == nil {
		if err := os.Remove(fntCachePath); err != nil {
			errors = append(errors, fmt.Sprintf("Failed to delete FNTCACHE.DAT: %s", err.Error()))
		} else {
			outputs = append(outputs, "Deleted FNTCACHE.DAT")
		}
	}

	// Restart Font Cache service
	out, err = exec.Command("net", "start", "FontCache").CombinedOutput()
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to start FontCache: %s - %s", err.Error(), strings.TrimSpace(string(out))))
	} else {
		outputs = append(outputs, "Restarted Font Cache service")
	}

	result.Output = strings.Join(outputs, "\n")
	result.Errors = errors
	result.Success = len(errors) == 0

	return result, nil
}

// ResetWindowsSearch stops the Windows Search service, deletes the search
// database, and restarts the service.
func ResetWindowsSearch() (*ToolResult, error) {
	result := &ToolResult{
		Name: "Reset Windows Search",
	}

	if !IsAdmin() {
		result.Success = false
		result.Output = "This tool requires administrator privileges. Please run CleanForge as administrator."
		result.Errors = append(result.Errors, "not running as administrator")
		return result, nil
	}

	var outputs []string
	var errors []string

	// Stop Windows Search service
	out, err := exec.Command("net", "stop", "WSearch").CombinedOutput()
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to stop WSearch: %s - %s", err.Error(), strings.TrimSpace(string(out))))
	} else {
		outputs = append(outputs, "Stopped Windows Search service")
	}

	// Delete search database
	programData := os.Getenv("ProgramData")
	searchDBPath := filepath.Join(programData, "Microsoft", "Search", "Data", "Applications", "Windows")

	if _, err := os.Stat(searchDBPath); err == nil {
		err = os.RemoveAll(searchDBPath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to delete search database: %s", err.Error()))
		} else {
			outputs = append(outputs, "Deleted Windows Search database")
		}
	} else {
		outputs = append(outputs, "Search database directory not found (may already be clean)")
	}

	// Restart Windows Search service
	out, err = exec.Command("net", "start", "WSearch").CombinedOutput()
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to start WSearch: %s - %s", err.Error(), strings.TrimSpace(string(out))))
	} else {
		outputs = append(outputs, "Restarted Windows Search service (re-indexing will begin automatically)")
	}

	result.Output = strings.Join(outputs, "\n")
	result.Errors = errors
	result.Success = len(errors) == 0

	return result, nil
}

// parseCSVLine splits a simple CSV line by commas, handling quoted fields.
func parseCSVLine(line string) []string {
	var fields []string
	var field strings.Builder
	inQuotes := false

	for i := 0; i < len(line); i++ {
		ch := line[i]
		switch {
		case ch == '"':
			if inQuotes && i+1 < len(line) && line[i+1] == '"' {
				// Escaped quote
				field.WriteByte('"')
				i++
			} else {
				inQuotes = !inQuotes
			}
		case ch == ',' && !inQuotes:
			fields = append(fields, field.String())
			field.Reset()
		default:
			field.WriteByte(ch)
		}
	}
	fields = append(fields, field.String())

	return fields
}

// friendlyName converts a package name like "Microsoft.BingWeather" into a
// more human-readable form like "Bing Weather".
func friendlyName(pkgName string) string {
	// Map of known package names to friendly names
	friendlyNames := map[string]string{
		"Microsoft.BingNews":                     "Bing News",
		"Microsoft.BingWeather":                  "Bing Weather",
		"Microsoft.GetHelp":                      "Get Help",
		"Microsoft.Getstarted":                   "Tips / Get Started",
		"Microsoft.MicrosoftOfficeHub":           "Office Hub",
		"Microsoft.MicrosoftSolitaireCollection": "Solitaire Collection",
		"Microsoft.People":                       "People",
		"Microsoft.WindowsFeedbackHub":           "Feedback Hub",
		"Microsoft.Xbox.TCUI":                    "Xbox TCUI",
		"Microsoft.XboxGameOverlay":              "Xbox Game Overlay",
		"Microsoft.XboxGamingOverlay":            "Xbox Game Bar",
		"Microsoft.XboxIdentityProvider":         "Xbox Identity Provider",
		"Microsoft.XboxSpeechToTextOverlay":      "Xbox Speech to Text",
		"Microsoft.YourPhone":                    "Your Phone / Phone Link",
		"Microsoft.ZuneMusic":                    "Groove Music",
		"Microsoft.ZuneVideo":                    "Movies & TV",
		"Clipchamp.Clipchamp":                    "Clipchamp",
		"Microsoft.Todos":                        "Microsoft To Do",
		"Microsoft.PowerAutomateDesktop":         "Power Automate Desktop",
		"Microsoft.549981C3F5F10":                "Cortana",
		"Disney.37853FC22B2CE":                   "Disney+",
		"SpotifyAB.SpotifyMusic":                 "Spotify",
		"king.com.CandyCrushSaga":                "Candy Crush Saga",
		"BytedancePte.Ltd.TikTok":                "TikTok",
	}

	if name, ok := friendlyNames[pkgName]; ok {
		return name
	}

	// Fallback: strip publisher prefix
	parts := strings.SplitN(pkgName, ".", 2)
	if len(parts) > 1 {
		return parts[1]
	}
	return pkgName
}

