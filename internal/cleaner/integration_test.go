//go:build integration

package cleaner

import (
	"os"
	"path/filepath"
	"testing"
)

// populateTempDir creates a temporary directory structure with test files
// mimicking various junk file categories. Returns the root temp dir path
// and a map of category names to their subdirectory paths.
func populateTempDir(t *testing.T) (string, map[string]string) {
	t.Helper()
	tmpDir := t.TempDir()

	dirs := map[string]string{
		"windows_temp": filepath.Join(tmpDir, "windows_temp"),
		"user_temp":    filepath.Join(tmpDir, "user_temp"),
		"chrome_cache": filepath.Join(tmpDir, "chrome_cache"),
		"npm_cache":    filepath.Join(tmpDir, "npm_cache"),
		"logs":         filepath.Join(tmpDir, "logs"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}
	}

	return tmpDir, dirs
}

// createTestFiles creates count files of the given size in the specified directory.
// Returns the total bytes written.
func createTestFiles(t *testing.T, dir string, count int, sizeBytes int) int64 {
	t.Helper()
	var total int64
	data := make([]byte, sizeBytes)
	for i := range data {
		data[i] = byte(i % 256)
	}

	for i := 0; i < count; i++ {
		name := filepath.Join(dir, "testfile_"+string(rune('a'+i%26))+string(rune('0'+i/26))+".tmp")
		if err := os.WriteFile(name, data, 0o644); err != nil {
			t.Fatalf("failed to create test file %s: %v", name, err)
		}
		total += int64(sizeBytes)
	}
	return total
}

// createTestFilesWithPrefix creates files with a unique prefix to avoid collisions.
func createTestFilesWithPrefix(t *testing.T, dir string, prefix string, count int, sizeBytes int) int64 {
	t.Helper()
	var total int64
	data := make([]byte, sizeBytes)
	for i := range data {
		data[i] = byte((i + 37) % 256)
	}

	for i := 0; i < count; i++ {
		name := filepath.Join(dir, prefix+"_"+itoa(i)+".tmp")
		if err := os.WriteFile(name, data, 0o644); err != nil {
			t.Fatalf("failed to create test file %s: %v", name, err)
		}
		total += int64(sizeBytes)
	}
	return total
}

// itoa is a simple int-to-string helper to avoid importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	result := ""
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	return result
}

// TestFullScanAndClean creates temp files, scans them via scanDirectory,
// cleans them via cleanPath, then verifies all files are removed and freed
// space matches the scan result.
func TestFullScanAndClean(t *testing.T) {
	_, dirs := populateTempDir(t)

	// Populate each category with test files
	expectedSizes := make(map[string]int64)
	expectedCounts := make(map[string]int)

	expectedSizes["windows_temp"] = createTestFiles(t, dirs["windows_temp"], 5, 1024)
	expectedCounts["windows_temp"] = 5

	expectedSizes["user_temp"] = createTestFiles(t, dirs["user_temp"], 3, 2048)
	expectedCounts["user_temp"] = 3

	expectedSizes["chrome_cache"] = createTestFiles(t, dirs["chrome_cache"], 4, 512)
	expectedCounts["chrome_cache"] = 4

	expectedSizes["npm_cache"] = createTestFiles(t, dirs["npm_cache"], 2, 4096)
	expectedCounts["npm_cache"] = 2

	expectedSizes["logs"] = createTestFiles(t, dirs["logs"], 3, 768)
	expectedCounts["logs"] = 3

	// Scan each directory and verify sizes match
	var totalScannedSize int64
	var totalScannedCount int

	for name, dir := range dirs {
		size, count, err := scanDirectory(dir)
		if err != nil {
			t.Errorf("scanDirectory(%s) returned error: %v", name, err)
			continue
		}

		if size != expectedSizes[name] {
			t.Errorf("scanDirectory(%s) size = %d, want %d", name, size, expectedSizes[name])
		}
		if count != expectedCounts[name] {
			t.Errorf("scanDirectory(%s) count = %d, want %d", name, count, expectedCounts[name])
		}

		totalScannedSize += size
		totalScannedCount += count
	}

	// Clean all directories
	var totalFreed int64
	var totalDeleted int

	for name, dir := range dirs {
		freed, deleted, errs := cleanPath(dir)
		if len(errs) > 0 {
			t.Errorf("cleanPath(%s) returned errors: %v", name, errs)
		}
		totalFreed += freed
		totalDeleted += deleted
	}

	// Verify freed space and deleted count match scan results
	if totalFreed != totalScannedSize {
		t.Errorf("total freed = %d, want %d (scan total)", totalFreed, totalScannedSize)
	}
	if totalDeleted != totalScannedCount {
		t.Errorf("total deleted = %d, want %d (scan total)", totalDeleted, totalScannedCount)
	}

	// Verify all files are actually gone
	for name, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Errorf("failed to read dir %s after clean: %v", name, err)
			continue
		}
		if len(entries) != 0 {
			t.Errorf("directory %s still has %d entries after clean, want 0", name, len(entries))
		}
	}
}

// TestScanThenPartialClean scans all directories but only cleans the "safe"
// ones, verifying that "low-risk" files still exist.
func TestScanThenPartialClean(t *testing.T) {
	_, dirs := populateTempDir(t)

	// "safe" categories
	createTestFiles(t, dirs["windows_temp"], 5, 1024)
	createTestFiles(t, dirs["user_temp"], 3, 1024)

	// "low-risk" categories
	createTestFiles(t, dirs["npm_cache"], 4, 1024)
	createTestFiles(t, dirs["logs"], 3, 1024)

	// Clean only "safe" categories
	safeDirs := []string{"windows_temp", "user_temp"}
	for _, name := range safeDirs {
		_, _, errs := cleanPath(dirs[name])
		if len(errs) > 0 {
			t.Errorf("cleanPath(%s) returned errors: %v", name, errs)
		}
	}

	// Verify safe directories are empty
	for _, name := range safeDirs {
		entries, err := os.ReadDir(dirs[name])
		if err != nil {
			t.Errorf("failed to read dir %s: %v", name, err)
			continue
		}
		if len(entries) != 0 {
			t.Errorf("safe dir %s still has %d entries after clean, want 0", name, len(entries))
		}
	}

	// Verify low-risk directories still have their files
	lowRiskDirs := []string{"npm_cache", "logs"}
	for _, name := range lowRiskDirs {
		entries, err := os.ReadDir(dirs[name])
		if err != nil {
			t.Errorf("failed to read dir %s: %v", name, err)
			continue
		}
		if len(entries) == 0 {
			t.Errorf("low-risk dir %s should still have files, but is empty", name)
		}
	}
}

// TestCleanIdempotent cleans a directory twice and verifies the second clean
// frees zero bytes.
func TestCleanIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	dir := filepath.Join(tmpDir, "idempotent")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	createTestFiles(t, dir, 10, 2048)

	// First clean
	freed1, deleted1, errs1 := cleanPath(dir)
	if len(errs1) > 0 {
		t.Errorf("first cleanPath returned errors: %v", errs1)
	}
	if freed1 == 0 {
		t.Error("first clean freed 0 bytes, expected > 0")
	}
	if deleted1 == 0 {
		t.Error("first clean deleted 0 files, expected > 0")
	}

	// Second clean should find nothing
	freed2, deleted2, errs2 := cleanPath(dir)
	if len(errs2) > 0 {
		t.Errorf("second cleanPath returned errors: %v", errs2)
	}
	if freed2 != 0 {
		t.Errorf("second clean freed %d bytes, want 0", freed2)
	}
	if deleted2 != 0 {
		t.Errorf("second clean deleted %d files, want 0", deleted2)
	}
}

// TestScanWithMixedPermissions creates some read-only files and verifies
// that scanDirectory still reports sizes correctly.
func TestScanWithMixedPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	dir := filepath.Join(tmpDir, "mixed_perms")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	// Create normal files
	normalSize := createTestFilesWithPrefix(t, dir, "normal", 5, 1024)

	// Create read-only files
	readOnlySize := int64(0)
	for i := 0; i < 3; i++ {
		name := filepath.Join(dir, "readonly_"+itoa(i)+".tmp")
		data := make([]byte, 512)
		if err := os.WriteFile(name, data, 0o444); err != nil {
			t.Fatalf("failed to create read-only file: %v", err)
		}
		readOnlySize += 512
	}

	expectedTotal := normalSize + readOnlySize
	expectedCount := 5 + 3

	size, count, err := scanDirectory(dir)
	if err != nil {
		t.Fatalf("scanDirectory returned error: %v", err)
	}

	if size != expectedTotal {
		t.Errorf("scanDirectory size = %d, want %d", size, expectedTotal)
	}
	if count != expectedCount {
		t.Errorf("scanDirectory count = %d, want %d", count, expectedCount)
	}
}

// TestLargeNumberOfFiles creates 1000+ small files and verifies scan and
// clean handle them correctly.
func TestLargeNumberOfFiles(t *testing.T) {
	tmpDir := t.TempDir()
	dir := filepath.Join(tmpDir, "large_set")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	const fileCount = 1100
	const fileSize = 128

	// Create files in batches to avoid filename collisions
	var expectedTotal int64
	data := make([]byte, fileSize)
	for i := range data {
		data[i] = byte(i % 256)
	}

	for i := 0; i < fileCount; i++ {
		name := filepath.Join(dir, "file_"+itoa(i)+".tmp")
		if err := os.WriteFile(name, data, 0o644); err != nil {
			t.Fatalf("failed to create file %d: %v", i, err)
		}
		expectedTotal += fileSize
	}

	// Scan
	size, count, err := scanDirectory(dir)
	if err != nil {
		t.Fatalf("scanDirectory returned error: %v", err)
	}

	if size != expectedTotal {
		t.Errorf("scanDirectory size = %d, want %d", size, expectedTotal)
	}
	if count != fileCount {
		t.Errorf("scanDirectory count = %d, want %d", count, fileCount)
	}

	// Clean
	freed, deleted, errs := cleanPath(dir)
	if len(errs) > 0 {
		t.Errorf("cleanPath returned errors: %v", errs)
	}
	if freed != expectedTotal {
		t.Errorf("cleanPath freed = %d, want %d", freed, expectedTotal)
	}
	if deleted != fileCount {
		t.Errorf("cleanPath deleted = %d, want %d", deleted, fileCount)
	}

	// Verify directory is empty
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read dir after clean: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("directory has %d entries after clean, want 0", len(entries))
	}
}

// TestScanNonExistentDirectory verifies scanDirectory returns zero for a
// path that does not exist.
func TestScanNonExistentDirectory(t *testing.T) {
	size, count, err := scanDirectory(filepath.Join(t.TempDir(), "does_not_exist"))
	if err != nil {
		t.Errorf("scanDirectory returned unexpected error for non-existent path: %v", err)
	}
	if size != 0 || count != 0 {
		t.Errorf("scanDirectory(%q) = (%d, %d), want (0, 0)", "does_not_exist", size, count)
	}
}

// TestCleanNonExistentDirectory verifies cleanPath returns zero for a
// path that does not exist.
func TestCleanNonExistentDirectory(t *testing.T) {
	freed, deleted, errs := cleanPath(filepath.Join(t.TempDir(), "does_not_exist"))
	if len(errs) > 0 {
		t.Errorf("cleanPath returned unexpected errors: %v", errs)
	}
	if freed != 0 || deleted != 0 {
		t.Errorf("cleanPath non-existent = (%d, %d), want (0, 0)", freed, deleted)
	}
}

// TestScanSingleFile verifies scanDirectory works when pointed at a single file
// rather than a directory.
func TestScanSingleFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "single_file.dat")

	data := make([]byte, 4096)
	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	size, count, err := scanDirectory(filePath)
	if err != nil {
		t.Fatalf("scanDirectory returned error: %v", err)
	}
	if size != 4096 {
		t.Errorf("scanDirectory single file size = %d, want 4096", size)
	}
	if count != 1 {
		t.Errorf("scanDirectory single file count = %d, want 1", count)
	}
}

// TestCleanSingleFile verifies cleanPath removes a single file.
func TestCleanSingleFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "single_file.dat")

	data := make([]byte, 2048)
	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	freed, deleted, errs := cleanPath(filePath)
	if len(errs) > 0 {
		t.Errorf("cleanPath returned errors: %v", errs)
	}
	if freed != 2048 {
		t.Errorf("cleanPath single file freed = %d, want 2048", freed)
	}
	if deleted != 1 {
		t.Errorf("cleanPath single file deleted = %d, want 1", deleted)
	}

	// File should be gone
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("file still exists after cleanPath")
	}
}

// TestScanNestedDirectories verifies scanDirectory recursively walks nested dirs.
func TestScanNestedDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	root := filepath.Join(tmpDir, "nested")

	// Create nested structure: root/a/b/c
	deepDir := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(deepDir, 0o755); err != nil {
		t.Fatalf("failed to create nested dirs: %v", err)
	}

	// Create files at each level
	levels := []string{root, filepath.Join(root, "a"), filepath.Join(root, "a", "b"), deepDir}
	var expectedSize int64
	var expectedCount int

	for i, dir := range levels {
		fileSize := (i + 1) * 256
		name := filepath.Join(dir, "level_"+itoa(i)+".dat")
		data := make([]byte, fileSize)
		if err := os.WriteFile(name, data, 0o644); err != nil {
			t.Fatalf("failed to create file at level %d: %v", i, err)
		}
		expectedSize += int64(fileSize)
		expectedCount++
	}

	size, count, err := scanDirectory(root)
	if err != nil {
		t.Fatalf("scanDirectory returned error: %v", err)
	}

	if size != expectedSize {
		t.Errorf("scanDirectory nested size = %d, want %d", size, expectedSize)
	}
	if count != expectedCount {
		t.Errorf("scanDirectory nested count = %d, want %d", count, expectedCount)
	}
}

// TestCleanerNewCreatesCategories verifies that NewCleaner initializes all
// expected category IDs.
func TestCleanerNewCreatesCategories(t *testing.T) {
	c := NewCleaner("testuser")

	expectedIDs := []string{
		"windows_temp", "user_temp", "recycle_bin",
		"browser_cache_chrome", "browser_cache_edge", "browser_cache_firefox",
		"npm_cache", "maven_cache", "gradle_cache", "go_cache",
		"windows_update", "windows_logs", "prefetch", "thumbnails",
	}

	if len(c.categories) != len(expectedIDs) {
		t.Errorf("NewCleaner has %d categories, want %d", len(c.categories), len(expectedIDs))
	}

	catMap := make(map[string]bool)
	for _, cat := range c.categories {
		catMap[cat.ID] = true
	}

	for _, id := range expectedIDs {
		if !catMap[id] {
			t.Errorf("missing category ID: %s", id)
		}
	}
}

// TestCleanEmptyCategoryIDs verifies that Clean with an empty slice is a no-op.
func TestCleanEmptyCategoryIDs(t *testing.T) {
	c := NewCleaner("testuser")
	result, err := c.Clean([]string{})
	if err != nil {
		t.Fatalf("Clean returned error: %v", err)
	}
	if result.FreedSpace != 0 {
		t.Errorf("Clean([]) freed %d bytes, want 0", result.FreedSpace)
	}
	if result.DeletedFiles != 0 {
		t.Errorf("Clean([]) deleted %d files, want 0", result.DeletedFiles)
	}
}

// TestResolveGlobPathsNoMatch verifies resolveGlobPaths returns empty for a
// pattern that matches nothing.
func TestResolveGlobPathsNoMatch(t *testing.T) {
	result := resolveGlobPaths(filepath.Join(t.TempDir(), "nonexistent_*.xyz"))
	if len(result) != 0 {
		t.Errorf("resolveGlobPaths returned %d matches, want 0", len(result))
	}
}

// TestResolveGlobPathsWithMatches verifies resolveGlobPaths returns correct
// matches.
func TestResolveGlobPathsWithMatches(t *testing.T) {
	tmpDir := t.TempDir()

	// Create matching files
	for i := 0; i < 3; i++ {
		name := filepath.Join(tmpDir, "match_"+itoa(i)+".log")
		if err := os.WriteFile(name, []byte("test"), 0o644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
	}

	// Create non-matching file
	if err := os.WriteFile(filepath.Join(tmpDir, "other.txt"), []byte("test"), 0o644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	result := resolveGlobPaths(filepath.Join(tmpDir, "match_*.log"))
	if len(result) != 3 {
		t.Errorf("resolveGlobPaths returned %d matches, want 3", len(result))
	}
}
