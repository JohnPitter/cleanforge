package cleaner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// CleanCategory represents a category of files that can be cleaned.
type CleanCategory struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Icon        string   `json:"icon"`
	Risk        string   `json:"risk"` // "safe", "low", "medium"
	Size        int64    `json:"size"`
	FileCount   int      `json:"fileCount"`
	Paths       []string `json:"-"`
}

// ScanResult contains the results of scanning all cleanup categories.
type ScanResult struct {
	Categories []CleanCategory `json:"categories"`
	TotalSize  int64           `json:"totalSize"`
	TotalFiles int             `json:"totalFiles"`
}

// CleanResult contains the results of a cleanup operation.
type CleanResult struct {
	FreedSpace   int64    `json:"freedSpace"`
	DeletedFiles int      `json:"deletedFiles"`
	Errors       []string `json:"errors"`
}

// CleanProgress reports progress during a cleanup operation.
type CleanProgress struct {
	Category    string  `json:"category"`
	Current     int     `json:"current"`
	Total       int     `json:"total"`
	Percentage  float64 `json:"percentage"`
	CurrentFile string  `json:"currentFile"`
}

// Cleaner manages file scanning and cleanup operations.
type Cleaner struct {
	username   string
	categories []CleanCategory
}

// NewCleaner creates a new Cleaner instance with all category definitions populated
// using the provided Windows username for user-specific paths.
func NewCleaner(username string) *Cleaner {
	userHome := filepath.Join("C:\\Users", username)
	appDataLocal := filepath.Join(userHome, "AppData", "Local")
	appDataRoaming := filepath.Join(userHome, "AppData", "Roaming")

	categories := []CleanCategory{
		{
			ID:          "windows_temp",
			Name:        "Windows Temp Files",
			Description: "Temporary files created by Windows and applications",
			Icon:        "trash",
			Risk:        "safe",
			Paths:       []string{`C:\Windows\Temp`},
		},
		{
			ID:          "user_temp",
			Name:        "User Temp Files",
			Description: "Temporary files in your user profile",
			Icon:        "trash",
			Risk:        "safe",
			Paths:       []string{filepath.Join(appDataLocal, "Temp")},
		},
		{
			ID:          "recycle_bin",
			Name:        "Recycle Bin",
			Description: "Files in the Windows Recycle Bin",
			Icon:        "recycle",
			Risk:        "safe",
			Paths:       []string{}, // handled via PowerShell
		},
		{
			ID:          "browser_cache_chrome",
			Name:        "Chrome Cache",
			Description: "Google Chrome browser cache files",
			Icon:        "globe",
			Risk:        "safe",
			Paths: []string{
				filepath.Join(appDataLocal, "Google", "Chrome", "User Data", "Default", "Cache"),
				filepath.Join(appDataLocal, "Google", "Chrome", "User Data", "Default", "Code Cache"),
			},
		},
		{
			ID:          "browser_cache_edge",
			Name:        "Edge Cache",
			Description: "Microsoft Edge browser cache files",
			Icon:        "globe",
			Risk:        "safe",
			Paths: []string{
				filepath.Join(appDataLocal, "Microsoft", "Edge", "User Data", "Default", "Cache"),
				filepath.Join(appDataLocal, "Microsoft", "Edge", "User Data", "Default", "Code Cache"),
			},
		},
		{
			ID:          "browser_cache_firefox",
			Name:        "Firefox Cache",
			Description: "Mozilla Firefox browser cache files",
			Icon:        "globe",
			Risk:        "safe",
			Paths:       resolveGlobPaths(filepath.Join(appDataLocal, "Mozilla", "Firefox", "Profiles", "*", "cache2")),
		},
		{
			ID:          "npm_cache",
			Name:        "npm Cache",
			Description: "Node.js package manager cache",
			Icon:        "package",
			Risk:        "low",
			Paths:       []string{filepath.Join(appDataRoaming, "npm-cache")},
		},
		{
			ID:          "maven_cache",
			Name:        "Maven Cache",
			Description: "Apache Maven local repository cache",
			Icon:        "package",
			Risk:        "low",
			Paths:       []string{filepath.Join(userHome, ".m2", "repository")},
		},
		{
			ID:          "gradle_cache",
			Name:        "Gradle Cache",
			Description: "Gradle build system cache",
			Icon:        "package",
			Risk:        "low",
			Paths:       []string{filepath.Join(userHome, ".gradle", "caches")},
		},
		{
			ID:          "go_cache",
			Name:        "Go Cache",
			Description: "Go build and module cache",
			Icon:        "package",
			Risk:        "low",
			Paths:       []string{}, // handled via `go clean -cache`
		},
		{
			ID:          "windows_update",
			Name:        "Windows Update Cache",
			Description: "Downloaded Windows Update files",
			Icon:        "download",
			Risk:        "low",
			Paths:       []string{`C:\Windows\SoftwareDistribution\Download`},
		},
		{
			ID:          "windows_logs",
			Name:        "Windows Logs",
			Description: "Windows system log files",
			Icon:        "file-text",
			Risk:        "low",
			Paths:       []string{`C:\Windows\Logs`},
		},
		{
			ID:          "prefetch",
			Name:        "Prefetch Files",
			Description: "Windows application prefetch data",
			Icon:        "zap",
			Risk:        "low",
			Paths:       []string{`C:\Windows\Prefetch`},
		},
		{
			ID:          "thumbnails",
			Name:        "Thumbnail Cache",
			Description: "Windows Explorer thumbnail cache files",
			Icon:        "image",
			Risk:        "safe",
			Paths:       resolveGlobPaths(filepath.Join(appDataLocal, "Microsoft", "Windows", "Explorer", "thumbcache_*.db")),
		},
	}

	return &Cleaner{
		username:   username,
		categories: categories,
	}
}

// Scan examines all cleanup categories and calculates the total size and file count
// for each category. Returns a ScanResult with the findings.
func (c *Cleaner) Scan() (*ScanResult, error) {
	result := &ScanResult{
		Categories: make([]CleanCategory, 0, len(c.categories)),
	}

	for _, cat := range c.categories {
		scannedCat := cat
		scannedCat.Size = 0
		scannedCat.FileCount = 0

		switch cat.ID {
		case "recycle_bin":
			size, count := scanRecycleBin()
			scannedCat.Size = size
			scannedCat.FileCount = count

		case "go_cache":
			size, count := scanGoCache()
			scannedCat.Size = size
			scannedCat.FileCount = count

		default:
			for _, p := range cat.Paths {
				size, count, err := scanDirectory(p)
				if err != nil {
					// Skip directories we can't access
					continue
				}
				scannedCat.Size += size
				scannedCat.FileCount += count
			}
		}

		result.Categories = append(result.Categories, scannedCat)
		result.TotalSize += scannedCat.Size
		result.TotalFiles += scannedCat.FileCount
	}

	return result, nil
}

// Clean deletes files in the specified category IDs. Pass an empty slice to skip cleaning.
// Returns a CleanResult with statistics about the operation.
func (c *Cleaner) Clean(categoryIDs []string) (*CleanResult, error) {
	if len(categoryIDs) == 0 {
		return &CleanResult{}, nil
	}

	// Build a lookup set for the requested categories
	requested := make(map[string]bool, len(categoryIDs))
	for _, id := range categoryIDs {
		requested[id] = true
	}

	result := &CleanResult{
		Errors: make([]string, 0),
	}

	for _, cat := range c.categories {
		if !requested[cat.ID] {
			continue
		}

		switch cat.ID {
		case "recycle_bin":
			freed, deleted, errs := cleanRecycleBin()
			result.FreedSpace += freed
			result.DeletedFiles += deleted
			result.Errors = append(result.Errors, errs...)

		case "go_cache":
			freed, deleted, errs := cleanGoCache()
			result.FreedSpace += freed
			result.DeletedFiles += deleted
			result.Errors = append(result.Errors, errs...)

		default:
			for _, p := range cat.Paths {
				freed, deleted, errs := cleanPath(p)
				result.FreedSpace += freed
				result.DeletedFiles += deleted
				result.Errors = append(result.Errors, errs...)
			}
		}
	}

	return result, nil
}

// scanDirectory recursively walks a directory and returns total size in bytes and file count.
// Permission errors and inaccessible files are silently skipped.
func scanDirectory(path string) (int64, int, error) {
	var totalSize int64
	var fileCount int

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, nil
		}
		return 0, 0, fmt.Errorf("cannot access %s: %w", path, err)
	}

	// If it's a single file (e.g. thumbcache_*.db matched as individual file)
	if !info.IsDir() {
		return info.Size(), 1, nil
	}

	err = filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip files/dirs we can't access
			return nil
		}
		if !info.IsDir() {
			totalSize += info.Size()
			fileCount++
		}
		return nil
	})

	if err != nil {
		return totalSize, fileCount, fmt.Errorf("error walking %s: %w", path, err)
	}

	return totalSize, fileCount, nil
}

// scanRecycleBin estimates the size and item count of the Windows Recycle Bin
// using a PowerShell command.
func scanRecycleBin() (int64, int) {
	// Get total size
	sizeOut, err := exec.Command("powershell", "-Command",
		"(New-Object -ComObject Shell.Application).NameSpace(10).Items() | Measure-Object -Property Size -Sum | Select-Object -ExpandProperty Sum").Output()
	if err != nil {
		return 0, 0
	}

	sizeStr := strings.TrimSpace(string(sizeOut))
	if sizeStr == "" {
		return 0, 0
	}

	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		return 0, 0
	}

	// Get item count
	countOut, err := exec.Command("powershell", "-Command",
		"(New-Object -ComObject Shell.Application).NameSpace(10).Items() | Measure-Object | Select-Object -ExpandProperty Count").Output()
	if err != nil {
		return size, 0
	}

	countStr := strings.TrimSpace(string(countOut))
	count, err := strconv.Atoi(countStr)
	if err != nil {
		return size, 0
	}

	return size, count
}

// scanGoCache estimates the size of the Go build cache by checking the cache directory.
func scanGoCache() (int64, int) {
	out, err := exec.Command("go", "env", "GOCACHE").Output()
	if err != nil {
		return 0, 0
	}

	cacheDir := strings.TrimSpace(string(out))
	if cacheDir == "" {
		return 0, 0
	}

	size, count, err := scanDirectory(cacheDir)
	if err != nil {
		return 0, 0
	}

	return size, count
}

// cleanRecycleBin empties the Windows Recycle Bin using PowerShell.
func cleanRecycleBin() (int64, int, []string) {
	var errs []string

	// Get current size before cleaning
	sizeBefore, countBefore := scanRecycleBin()

	err := exec.Command("powershell", "-Command",
		"Clear-RecycleBin -Force -ErrorAction SilentlyContinue").Run()
	if err != nil {
		errs = append(errs, fmt.Sprintf("recycle_bin: failed to clear recycle bin: %v", err))
		return 0, 0, errs
	}

	return sizeBefore, countBefore, errs
}

// cleanGoCache runs `go clean -cache` to clear the Go build cache.
func cleanGoCache() (int64, int, []string) {
	var errs []string

	// Get current size before cleaning
	sizeBefore, countBefore := scanGoCache()

	err := exec.Command("go", "clean", "-cache").Run()
	if err != nil {
		errs = append(errs, fmt.Sprintf("go_cache: failed to clean go cache: %v", err))
		return 0, 0, errs
	}

	return sizeBefore, countBefore, errs
}

// cleanPath deletes all files and subdirectories within a given path.
// The top-level directory itself is preserved. Permission errors are collected
// but do not stop the operation.
func cleanPath(path string) (int64, int, []string) {
	var freedSpace int64
	var deletedFiles int
	var errs []string

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, nil
		}
		errs = append(errs, fmt.Sprintf("%s: cannot access: %v", path, err))
		return 0, 0, errs
	}

	// If it's a single file, delete it directly
	if !info.IsDir() {
		size := info.Size()
		err := os.Remove(path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", path, err))
			return 0, 0, errs
		}
		return size, 1, errs
	}

	// Collect all entries in the directory
	entries, err := os.ReadDir(path)
	if err != nil {
		errs = append(errs, fmt.Sprintf("%s: cannot read directory: %v", path, err))
		return 0, 0, errs
	}

	for _, entry := range entries {
		entryPath := filepath.Join(path, entry.Name())

		if entry.IsDir() {
			// Calculate size before removal
			size, count, _ := scanDirectory(entryPath)
			err := os.RemoveAll(entryPath)
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", entryPath, err))
				continue
			}
			freedSpace += size
			deletedFiles += count
		} else {
			entryInfo, err := entry.Info()
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s: cannot get file info: %v", entryPath, err))
				continue
			}
			size := entryInfo.Size()
			err = os.Remove(entryPath)
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", entryPath, err))
				continue
			}
			freedSpace += size
			deletedFiles++
		}
	}

	return freedSpace, deletedFiles, errs
}

// resolveGlobPaths expands a glob pattern into matching file paths.
// Returns an empty slice if no matches are found or an error occurs.
func resolveGlobPaths(pattern string) []string {
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return []string{}
	}
	return matches
}
