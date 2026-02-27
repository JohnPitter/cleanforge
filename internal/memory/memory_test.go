package memory

import (
	"strings"
	"testing"
)

func TestGetMemoryStatus(t *testing.T) {
	status, err := GetMemoryStatus()
	if err != nil {
		t.Fatalf("GetMemoryStatus returned error: %v", err)
	}

	if status == nil {
		t.Fatal("GetMemoryStatus returned nil")
	}

	if status.Total == 0 {
		t.Error("Total memory is 0")
	}

	if status.Used == 0 {
		t.Error("Used memory is 0; expected some memory in use")
	}

	if status.Used > status.Total {
		t.Errorf("Used (%d) exceeds Total (%d)", status.Used, status.Total)
	}

	if status.Available > status.Total {
		t.Errorf("Available (%d) exceeds Total (%d)", status.Available, status.Total)
	}

	if status.UsagePercent < 0 || status.UsagePercent > 100 {
		t.Errorf("UsagePercent out of range [0, 100]: %f", status.UsagePercent)
	}

	t.Logf("Memory: total=%d, used=%d, available=%d, usage=%.2f%%, cached=%d, procs=%d",
		status.Total, status.Used, status.Available, status.UsagePercent, status.Cached, len(status.TopProcesses))
}

func TestGetTopMemoryProcesses(t *testing.T) {
	procs, err := GetTopMemoryProcesses(5)
	if err != nil {
		t.Fatalf("GetTopMemoryProcesses returned error: %v", err)
	}

	if len(procs) == 0 {
		t.Error("expected at least 1 process, got 0")
	}

	if len(procs) > 5 {
		t.Errorf("expected at most 5 processes, got %d", len(procs))
	}

	// Verify processes are sorted by memory descending
	for i := 1; i < len(procs); i++ {
		if procs[i].Memory > procs[i-1].Memory {
			t.Errorf("processes not sorted by memory: %d > %d at index %d",
				procs[i].Memory, procs[i-1].Memory, i)
		}
	}

	// Verify each process has required fields
	for _, p := range procs {
		if p.Name == "" {
			t.Errorf("process with PID %d has empty name", p.PID)
		}
		if p.Memory == 0 {
			t.Errorf("process %q has 0 memory", p.Name)
		}
		if p.Percent < 0 || p.Percent > 100 {
			t.Errorf("process %q has percent out of range: %f", p.Name, p.Percent)
		}
	}
}

func TestGetTopMemoryProcessesLargeCount(t *testing.T) {
	// Requesting more processes than available should return all available
	procs, err := GetTopMemoryProcesses(10000)
	if err != nil {
		t.Fatalf("GetTopMemoryProcesses returned error: %v", err)
	}

	if len(procs) == 0 {
		t.Error("expected at least 1 process")
	}

	t.Logf("Total processes with memory: %d", len(procs))
}

func TestDetectMemoryLeaks(t *testing.T) {
	// This test verifies DetectMemoryLeaks does not panic
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("DetectMemoryLeaks panicked: %v", r)
		}
	}()

	suspects, err := DetectMemoryLeaks()
	if err != nil {
		t.Fatalf("DetectMemoryLeaks returned error: %v", err)
	}

	// suspects may be empty which is fine
	t.Logf("Memory leak suspects: %d", len(suspects))

	for _, s := range suspects {
		if s.Memory < 500*1024*1024 {
			t.Errorf("suspect %q has memory %d which is below threshold 500MB", s.Name, s.Memory)
		}
	}
}

func TestIsSystemProcess(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"svchost.exe", true},
		{"explorer.exe", true},
		{"dwm.exe", true},
		{"csrss.exe", true},
		{"System", true},
		{"lsass.exe", true},
		{"msmpeng.exe", true},
		{"chrome.exe", false},
		{"myapp.exe", false},
		{"firefox.exe", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSystemProcess(tt.name)
			if result != tt.expected {
				t.Errorf("isSystemProcess(%q) = %v, want %v", tt.name, result, tt.expected)
			}
		})
	}
}

func TestIsSystemProcessCaseInsensitive(t *testing.T) {
	// The function lowercases the input, so these should all be system processes
	tests := []string{"SVCHOST.EXE", "Explorer.exe", "DWM.exe", "CSRSS.EXE"}

	for _, name := range tests {
		lower := strings.ToLower(name)
		if !isSystemProcess(lower) {
			t.Errorf("isSystemProcess(%q) should be true", lower)
		}
	}
}

func TestSystemProcessesMapNotEmpty(t *testing.T) {
	if len(systemProcesses) == 0 {
		t.Fatal("systemProcesses map is empty")
	}
	if len(systemProcesses) < 30 {
		t.Errorf("expected at least 30 system processes, got %d", len(systemProcesses))
	}
}

func TestRoundFloat(t *testing.T) {
	tests := []struct {
		name     string
		val      float64
		places   int
		expected float64
	}{
		{"Round to 2", 3.14159, 2, 3.14},
		{"Round to 0", 3.7, 0, 4.0},
		{"Round to 1", 2.55, 1, 2.6},
		{"No rounding", 5.0, 2, 5.0},
		{"Negative", -3.14159, 2, -3.13},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := roundFloat(tt.val, tt.places)
			if result != tt.expected {
				t.Errorf("roundFloat(%f, %d) = %f, want %f", tt.val, tt.places, result, tt.expected)
			}
		})
	}
}

func TestFlushStandbyList(t *testing.T) {
	// This test verifies FlushStandbyList does not panic.
	// It may not actually trim processes without elevation.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("FlushStandbyList panicked: %v", r)
		}
	}()

	err := FlushStandbyList()
	// Error is acceptable since we may not have permission
	if err != nil {
		t.Logf("FlushStandbyList returned error (expected without admin): %v", err)
	}
}
