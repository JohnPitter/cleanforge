package cleaner

import (
	"os"
	"path/filepath"
	"testing"
)

// createTempFiles is a helper that creates a temporary directory containing
// the specified number of files, each of the given size. It returns the
// directory path and the total size of all files created.
func createTempFiles(t *testing.T, count int, sizePerFile int64) string {
	t.Helper()
	dir := t.TempDir()

	for i := 0; i < count; i++ {
		name := filepath.Join(dir, "file_"+string(rune('a'+i))+".tmp")
		data := make([]byte, sizePerFile)
		if err := os.WriteFile(name, data, 0644); err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
	}

	return dir
}

// createTempDirWithSubdirs creates a temporary directory that contains both
// files and a subdirectory with files.
func createTempDirWithSubdirs(t *testing.T) (string, int64, int) {
	t.Helper()
	dir := t.TempDir()

	// Create 3 files in the root directory
	for i := 0; i < 3; i++ {
		name := filepath.Join(dir, "root_file_"+string(rune('a'+i))+".tmp")
		data := make([]byte, 100)
		if err := os.WriteFile(name, data, 0644); err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
	}

	// Create a subdirectory with 2 files
	subDir := filepath.Join(dir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}
	for i := 0; i < 2; i++ {
		name := filepath.Join(subDir, "sub_file_"+string(rune('a'+i))+".tmp")
		data := make([]byte, 200)
		if err := os.WriteFile(name, data, 0644); err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
	}

	// Total: 3*100 + 2*200 = 700 bytes, 5 files
	return dir, 700, 5
}

func TestNewCleaner(t *testing.T) {
	c := NewCleaner("TestUser")

	if c == nil {
		t.Fatal("NewCleaner returned nil")
	}

	if c.username != "TestUser" {
		t.Errorf("expected username %q, got %q", "TestUser", c.username)
	}

	if len(c.categories) == 0 {
		t.Fatal("NewCleaner created no categories")
	}

	t.Run("HasExpectedCategories", func(t *testing.T) {
		expectedIDs := map[string]bool{
			"windows_temp":          true,
			"user_temp":             true,
			"recycle_bin":           true,
			"browser_cache_chrome":  true,
			"browser_cache_edge":    true,
			"browser_cache_firefox": true,
			"npm_cache":             true,
			"maven_cache":           true,
			"gradle_cache":          true,
			"go_cache":              true,
			"windows_update":        true,
			"windows_logs":          true,
			"prefetch":              true,
			"thumbnails":            true,
		}

		foundIDs := make(map[string]bool)
		for _, cat := range c.categories {
			foundIDs[cat.ID] = true
		}

		for id := range expectedIDs {
			if !foundIDs[id] {
				t.Errorf("expected category %q not found", id)
			}
		}
	})

	t.Run("UserTempPathContainsUsername", func(t *testing.T) {
		for _, cat := range c.categories {
			if cat.ID == "user_temp" {
				if len(cat.Paths) == 0 {
					t.Error("user_temp category has no paths")
					return
				}
				found := false
				for _, p := range cat.Paths {
					if filepath.Base(filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(p))))) == "TestUser" ||
						// Check if the path contains TestUser anywhere
						containsPath(p, "TestUser") {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("user_temp path does not contain username; paths: %v", cat.Paths)
				}
				return
			}
		}
		t.Error("user_temp category not found")
	})
}

func containsPath(path, segment string) bool {
	for _, part := range filepath.SplitList(path) {
		if part == segment {
			return true
		}
	}
	// Fallback: check if the segment appears in the path string
	return filepath.Clean(path) != "" && len(path) > 0 && findInPath(path, segment)
}

func findInPath(path, segment string) bool {
	dir := path
	for {
		base := filepath.Base(dir)
		if base == segment {
			return true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return false
}

func TestScanDirectory(t *testing.T) {
	dir := createTempFiles(t, 5, 1024)

	size, count, err := scanDirectory(dir)
	if err != nil {
		t.Fatalf("scanDirectory returned error: %v", err)
	}

	if count != 5 {
		t.Errorf("expected 5 files, got %d", count)
	}

	expectedSize := int64(5 * 1024)
	if size != expectedSize {
		t.Errorf("expected total size %d, got %d", expectedSize, size)
	}
}

func TestScanDirectoryWithSubdirs(t *testing.T) {
	dir, expectedSize, expectedCount := createTempDirWithSubdirs(t)

	size, count, err := scanDirectory(dir)
	if err != nil {
		t.Fatalf("scanDirectory returned error: %v", err)
	}

	if count != expectedCount {
		t.Errorf("expected %d files, got %d", expectedCount, count)
	}

	if size != expectedSize {
		t.Errorf("expected total size %d, got %d", expectedSize, size)
	}
}

func TestScanDirectoryEmpty(t *testing.T) {
	dir := t.TempDir()

	size, count, err := scanDirectory(dir)
	if err != nil {
		t.Fatalf("scanDirectory returned error on empty dir: %v", err)
	}

	if size != 0 {
		t.Errorf("expected size 0 for empty dir, got %d", size)
	}

	if count != 0 {
		t.Errorf("expected count 0 for empty dir, got %d", count)
	}
}

func TestScanDirectoryNotExist(t *testing.T) {
	nonExistentPath := filepath.Join(t.TempDir(), "does_not_exist")

	size, count, err := scanDirectory(nonExistentPath)
	if err != nil {
		t.Fatalf("scanDirectory should not return error for non-existent dir, got: %v", err)
	}

	if size != 0 {
		t.Errorf("expected size 0 for non-existent dir, got %d", size)
	}

	if count != 0 {
		t.Errorf("expected count 0 for non-existent dir, got %d", count)
	}
}

func TestScanDirectorySingleFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "single.txt")
	data := make([]byte, 512)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	size, count, err := scanDirectory(filePath)
	if err != nil {
		t.Fatalf("scanDirectory returned error for single file: %v", err)
	}

	if count != 1 {
		t.Errorf("expected 1 file, got %d", count)
	}

	if size != 512 {
		t.Errorf("expected size 512, got %d", size)
	}
}

func TestCleanPath(t *testing.T) {
	dir := createTempFiles(t, 3, 256)

	freed, deleted, errs := cleanPath(dir)
	if len(errs) > 0 {
		t.Logf("cleanPath reported errors (may be expected): %v", errs)
	}

	if deleted != 3 {
		t.Errorf("expected 3 deleted files, got %d", deleted)
	}

	expectedFreed := int64(3 * 256)
	if freed != expectedFreed {
		t.Errorf("expected %d freed bytes, got %d", expectedFreed, freed)
	}

	// Verify the directory itself still exists
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("directory should still exist after cleaning: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected the path to still be a directory")
	}

	// Verify the directory is now empty
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read cleaned directory: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected empty directory after clean, got %d entries", len(entries))
	}
}

func TestCleanPathNonExistent(t *testing.T) {
	nonExistentPath := filepath.Join(t.TempDir(), "does_not_exist")

	freed, deleted, errs := cleanPath(nonExistentPath)
	if len(errs) != 0 {
		t.Errorf("expected no errors for non-existent path, got: %v", errs)
	}

	if freed != 0 || deleted != 0 {
		t.Errorf("expected 0 freed and 0 deleted for non-existent path, got freed=%d deleted=%d", freed, deleted)
	}
}

func TestCleanPathSingleFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "to_delete.txt")
	data := make([]byte, 128)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	freed, deleted, errs := cleanPath(filePath)
	if len(errs) > 0 {
		t.Errorf("unexpected errors: %v", errs)
	}

	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}

	if freed != 128 {
		t.Errorf("expected 128 freed, got %d", freed)
	}

	// File should be gone
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("file should have been deleted")
	}
}

func TestScan(t *testing.T) {
	// Create a Cleaner with temp directories containing known files
	dir1 := createTempFiles(t, 2, 100)
	dir2 := createTempFiles(t, 3, 200)

	c := &Cleaner{
		username: "test",
		categories: []CleanCategory{
			{
				ID:    "test_cat1",
				Name:  "Test Category 1",
				Paths: []string{dir1},
			},
			{
				ID:    "test_cat2",
				Name:  "Test Category 2",
				Paths: []string{dir2},
			},
		},
	}

	result, err := c.Scan()
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}

	if result == nil {
		t.Fatal("Scan returned nil result")
	}

	if len(result.Categories) != 2 {
		t.Errorf("expected 2 categories, got %d", len(result.Categories))
	}

	expectedTotalSize := int64(2*100 + 3*200)
	if result.TotalSize != expectedTotalSize {
		t.Errorf("expected total size %d, got %d", expectedTotalSize, result.TotalSize)
	}

	expectedTotalFiles := 5
	if result.TotalFiles != expectedTotalFiles {
		t.Errorf("expected total files %d, got %d", expectedTotalFiles, result.TotalFiles)
	}

	// Verify individual category sizes
	for _, cat := range result.Categories {
		switch cat.ID {
		case "test_cat1":
			if cat.Size != 200 {
				t.Errorf("test_cat1 size: expected 200, got %d", cat.Size)
			}
			if cat.FileCount != 2 {
				t.Errorf("test_cat1 file count: expected 2, got %d", cat.FileCount)
			}
		case "test_cat2":
			if cat.Size != 600 {
				t.Errorf("test_cat2 size: expected 600, got %d", cat.Size)
			}
			if cat.FileCount != 3 {
				t.Errorf("test_cat2 file count: expected 3, got %d", cat.FileCount)
			}
		}
	}
}

func TestCleanSelectiveCategories(t *testing.T) {
	dir1 := createTempFiles(t, 2, 100)
	dir2 := createTempFiles(t, 3, 200)
	dir3 := createTempFiles(t, 1, 300)

	c := &Cleaner{
		username: "test",
		categories: []CleanCategory{
			{ID: "cat_a", Name: "Category A", Paths: []string{dir1}},
			{ID: "cat_b", Name: "Category B", Paths: []string{dir2}},
			{ID: "cat_c", Name: "Category C", Paths: []string{dir3}},
		},
	}

	// Only clean cat_a and cat_c, leave cat_b untouched
	result, err := c.Clean([]string{"cat_a", "cat_c"})
	if err != nil {
		t.Fatalf("Clean returned error: %v", err)
	}

	// cat_a: 2*100 = 200 bytes, 2 files
	// cat_c: 1*300 = 300 bytes, 1 file
	expectedFreed := int64(200 + 300)
	expectedDeleted := 3
	if result.FreedSpace != expectedFreed {
		t.Errorf("expected freed %d, got %d", expectedFreed, result.FreedSpace)
	}
	if result.DeletedFiles != expectedDeleted {
		t.Errorf("expected deleted %d, got %d", expectedDeleted, result.DeletedFiles)
	}

	// Verify cat_b directory still has its files
	entries, err := os.ReadDir(dir2)
	if err != nil {
		t.Fatalf("failed to read cat_b directory: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("cat_b should still have 3 files, has %d", len(entries))
	}
}

func TestCleanEmptyCategoryList(t *testing.T) {
	c := &Cleaner{
		username: "test",
		categories: []CleanCategory{
			{ID: "cat_a", Name: "Category A", Paths: []string{t.TempDir()}},
		},
	}

	result, err := c.Clean([]string{})
	if err != nil {
		t.Fatalf("Clean with empty list returned error: %v", err)
	}

	if result.FreedSpace != 0 || result.DeletedFiles != 0 {
		t.Errorf("expected no changes with empty category list, got freed=%d deleted=%d",
			result.FreedSpace, result.DeletedFiles)
	}
}

func TestCleanPermissionError(t *testing.T) {
	// Create a directory with a read-only file to simulate permission issues
	dir := t.TempDir()
	filePath := filepath.Join(dir, "readonly.txt")
	data := make([]byte, 64)
	if err := os.WriteFile(filePath, data, 0444); err != nil {
		t.Fatalf("failed to create readonly file: %v", err)
	}

	// Make the file writable for cleanup by t.TempDir
	t.Cleanup(func() {
		_ = os.Chmod(filePath, 0644)
	})

	c := &Cleaner{
		username: "test",
		categories: []CleanCategory{
			{ID: "test_perm", Name: "Permission Test", Paths: []string{dir}},
		},
	}

	// On Windows, read-only files can still sometimes be deleted,
	// but the important thing is that the operation does not panic or return a fatal error
	result, err := c.Clean([]string{"test_perm"})
	if err != nil {
		t.Fatalf("Clean should not return a fatal error on permission issues: %v", err)
	}

	// The result should exist regardless
	if result == nil {
		t.Fatal("Clean returned nil result")
	}

	// Errors are collected, not fatal
	t.Logf("Errors (may be empty on Windows): %v", result.Errors)
}

func TestResolveGlobPaths(t *testing.T) {
	t.Run("NoMatch", func(t *testing.T) {
		result := resolveGlobPaths(filepath.Join(t.TempDir(), "nonexistent_*_pattern"))
		if len(result) != 0 {
			t.Errorf("expected empty slice for no matches, got %v", result)
		}
	})

	t.Run("WithMatches", func(t *testing.T) {
		dir := t.TempDir()
		// Create files matching a pattern
		for _, name := range []string{"test_a.txt", "test_b.txt", "other.dat"} {
			if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0644); err != nil {
				t.Fatalf("failed to create file: %v", err)
			}
		}

		result := resolveGlobPaths(filepath.Join(dir, "test_*.txt"))
		if len(result) != 2 {
			t.Errorf("expected 2 matches, got %d: %v", len(result), result)
		}
	})
}
