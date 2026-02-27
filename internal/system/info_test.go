package system

import (
	"testing"
)

func TestGetDiskUsage(t *testing.T) {
	disks, err := GetDiskUsage()
	if err != nil {
		t.Fatalf("GetDiskUsage returned error: %v", err)
	}

	if len(disks) == 0 {
		t.Fatal("GetDiskUsage returned no disks; expected at least one on Windows")
	}

	for _, d := range disks {
		t.Run("Disk_"+d.Drive, func(t *testing.T) {
			if d.Total == 0 {
				t.Errorf("disk %s has 0 total size", d.Drive)
			}
			if d.UsagePercent < 0 || d.UsagePercent > 100 {
				t.Errorf("disk %s usage percent out of range: %f", d.Drive, d.UsagePercent)
			}
			if d.Free > d.Total {
				t.Errorf("disk %s free (%d) exceeds total (%d)", d.Drive, d.Free, d.Total)
			}
		})
	}
}

func TestGetCPUUsage(t *testing.T) {
	usage, err := GetCPUUsage()
	if err != nil {
		t.Fatalf("GetCPUUsage returned error: %v", err)
	}

	if usage < 0 || usage > 100 {
		t.Errorf("CPU usage out of range [0, 100]: got %f", usage)
	}
}

func TestGetRAMUsage(t *testing.T) {
	ram, err := GetRAMUsage()
	if err != nil {
		t.Fatalf("GetRAMUsage returned error: %v", err)
	}

	if ram == nil {
		t.Fatal("GetRAMUsage returned nil")
	}

	if ram.Total == 0 {
		t.Error("RAM total is 0")
	}

	if ram.Used > ram.Total {
		t.Errorf("RAM used (%d) exceeds total (%d)", ram.Used, ram.Total)
	}

	if ram.UsedPercent < 0 || ram.UsedPercent > 100 {
		t.Errorf("RAM usage percent out of range [0, 100]: got %f", ram.UsedPercent)
	}
}

func TestCalculateHealthScore(t *testing.T) {
	tests := []struct {
		name     string
		info     *SystemInfo
		minScore int
		maxScore int
	}{
		{
			name: "Perfect system - low CPU, low RAM, lots of free space",
			info: &SystemInfo{
				CPUUsage: 10,
				RAMUsage: 30,
				Disks: []DiskInfo{
					{Drive: "C:", UsagePercent: 40, Total: 500_000_000_000, Free: 300_000_000_000},
				},
			},
			minScore: 80,
			maxScore: 100,
		},
		{
			name: "Bad system - high CPU, high RAM, low disk",
			info: &SystemInfo{
				CPUUsage: 95,
				RAMUsage: 95,
				Disks: []DiskInfo{
					{Drive: "C:", UsagePercent: 95, Total: 500_000_000_000, Free: 25_000_000_000},
				},
			},
			minScore: 0,
			maxScore: 35,
		},
		{
			name: "Zero CPU usage",
			info: &SystemInfo{
				CPUUsage: 0,
				RAMUsage: 50,
				Disks: []DiskInfo{
					{Drive: "C:", UsagePercent: 50, Total: 500_000_000_000, Free: 250_000_000_000},
				},
			},
			minScore: 70,
			maxScore: 100,
		},
		{
			name: "100 percent RAM usage",
			info: &SystemInfo{
				CPUUsage: 20,
				RAMUsage: 100,
				Disks: []DiskInfo{
					{Drive: "C:", UsagePercent: 50, Total: 500_000_000_000, Free: 250_000_000_000},
				},
			},
			minScore: 50,
			maxScore: 80,
		},
		{
			name: "No disks reported",
			info: &SystemInfo{
				CPUUsage: 10,
				RAMUsage: 30,
				Disks:    []DiskInfo{},
			},
			minScore: 80,
			maxScore: 100,
		},
		{
			name: "Moderate CPU and RAM",
			info: &SystemInfo{
				CPUUsage: 60,
				RAMUsage: 75,
				Disks: []DiskInfo{
					{Drive: "C:", UsagePercent: 50, Total: 500_000_000_000, Free: 250_000_000_000},
				},
			},
			minScore: 50,
			maxScore: 80,
		},
		{
			name: "Critically low disk space under 10 percent free",
			info: &SystemInfo{
				CPUUsage: 10,
				RAMUsage: 30,
				Disks: []DiskInfo{
					{Drive: "C:", UsagePercent: 95, Total: 500_000_000_000, Free: 25_000_000_000},
				},
			},
			minScore: 55,
			maxScore: 80,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := CalculateHealthScore(tt.info)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("CalculateHealthScore = %d, expected between %d and %d", score, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestCalculateHealthScoreNeverNegative(t *testing.T) {
	// Worst possible scenario: every penalty hits
	info := &SystemInfo{
		CPUUsage: 100,
		RAMUsage: 100,
		Disks: []DiskInfo{
			{Drive: "C:", UsagePercent: 99, Total: 500_000_000_000, Free: 5_000_000_000},
		},
	}
	score := CalculateHealthScore(info)
	if score < 0 {
		t.Errorf("Health score should never be negative, got %d", score)
	}
}

func TestFormatUptime(t *testing.T) {
	tests := []struct {
		name     string
		seconds  uint64
		expected string
	}{
		{"Zero seconds", 0, "0m"},
		{"Only minutes", 300, "5m"},
		{"Hours and minutes", 3661, "1h 1m"},
		{"Days hours minutes", 90061, "1d 1h 1m"},
		{"Multiple days", 259200, "3d 0h 0m"},
		{"Large uptime", 1_000_000, "11d 13h 46m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatUptime(tt.seconds)
			if result != tt.expected {
				t.Errorf("formatUptime(%d) = %q, want %q", tt.seconds, result, tt.expected)
			}
		})
	}
}

func TestGetGPUInfo(t *testing.T) {
	// This test verifies GetGPUInfo does not panic.
	// On a CI or VM environment it may return empty strings.
	t.Run("DoesNotPanic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("GetGPUInfo panicked: %v", r)
			}
		}()

		name, driver := GetGPUInfo()
		// Logging the results for informational purposes; not asserting non-empty
		// since CI environments may not have a GPU.
		t.Logf("GPU Name: %q, Driver: %q", name, driver)
	})
}

func TestMathRound(t *testing.T) {
	tests := []struct {
		name     string
		val      float64
		places   int
		expected float64
	}{
		{"Round to 2 places", 3.14159, 2, 3.14},
		{"Round to 0 places", 3.7, 0, 4.0},
		{"Round to 1 place", 2.55, 1, 2.6},
		{"No rounding needed", 5.0, 2, 5.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := math_round(tt.val, tt.places)
			if result != tt.expected {
				t.Errorf("math_round(%f, %d) = %f, want %f", tt.val, tt.places, result, tt.expected)
			}
		})
	}
}
