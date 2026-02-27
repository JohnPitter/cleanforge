package monitor

import (
	"testing"
	"time"
)

func TestGetSnapshot(t *testing.T) {
	snapshot, err := GetSnapshot()
	if err != nil {
		t.Fatalf("GetSnapshot returned error: %v", err)
	}

	if snapshot == nil {
		t.Fatal("GetSnapshot returned nil")
	}

	if snapshot.Timestamp == 0 {
		t.Error("Timestamp is 0")
	}

	now := time.Now().Unix()
	if snapshot.Timestamp < now-10 || snapshot.Timestamp > now+10 {
		t.Errorf("Timestamp %d is too far from current time %d", snapshot.Timestamp, now)
	}

	if snapshot.CPUUsage < 0 || snapshot.CPUUsage > 100 {
		t.Errorf("CPUUsage out of range [0, 100]: %f", snapshot.CPUUsage)
	}

	if snapshot.RAMUsage < 0 || snapshot.RAMUsage > 100 {
		t.Errorf("RAMUsage out of range [0, 100]: %f", snapshot.RAMUsage)
	}

	if snapshot.DiskUsage < 0 || snapshot.DiskUsage > 100 {
		t.Errorf("DiskUsage out of range [0, 100]: %f", snapshot.DiskUsage)
	}

	t.Logf("Snapshot: cpu=%.1f%%, ram=%.1f%%, disk=%.1f%%, cpuTemp=%.1f, gpuTemp=%.1f, fan=%d",
		snapshot.CPUUsage, snapshot.RAMUsage, snapshot.DiskUsage,
		snapshot.CPUTemp, snapshot.GPUTemp, snapshot.FanSpeed)
}

func TestGetCPUTemp(t *testing.T) {
	// This may fail on machines without thermal sensors accessible to WMI
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("GetCPUTemp panicked: %v", r)
		}
	}()

	temp, err := GetCPUTemp()
	if err != nil {
		t.Logf("GetCPUTemp returned error (may be expected): %v", err)
		return
	}

	if temp < 0 || temp > 150 {
		t.Errorf("CPU temperature out of reasonable range: %f", temp)
	}

	t.Logf("CPU Temperature: %.1f°C", temp)
}

func TestGetGPUTemp(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("GetGPUTemp panicked: %v", r)
		}
	}()

	temp, err := GetGPUTemp()
	if err != nil {
		t.Logf("GetGPUTemp returned error (may be expected): %v", err)
		return
	}

	if temp < 0 || temp > 150 {
		t.Errorf("GPU temperature out of reasonable range: %f", temp)
	}

	t.Logf("GPU Temperature: %.1f°C", temp)
}

func TestDurationToScore(t *testing.T) {
	tests := []struct {
		name      string
		elapsed   time.Duration
		bestTime  time.Duration
		worstTime time.Duration
		expected  int
	}{
		{
			name:      "Best time gets 100",
			elapsed:   5 * time.Millisecond,
			bestTime:  10 * time.Millisecond,
			worstTime: 100 * time.Millisecond,
			expected:  100,
		},
		{
			name:      "Worst time gets 0",
			elapsed:   200 * time.Millisecond,
			bestTime:  10 * time.Millisecond,
			worstTime: 100 * time.Millisecond,
			expected:  0,
		},
		{
			name:      "Exactly best time",
			elapsed:   10 * time.Millisecond,
			bestTime:  10 * time.Millisecond,
			worstTime: 100 * time.Millisecond,
			expected:  100,
		},
		{
			name:      "Exactly worst time",
			elapsed:   100 * time.Millisecond,
			bestTime:  10 * time.Millisecond,
			worstTime: 100 * time.Millisecond,
			expected:  0,
		},
		{
			name:      "Midpoint gets ~50",
			elapsed:   55 * time.Millisecond,
			bestTime:  10 * time.Millisecond,
			worstTime: 100 * time.Millisecond,
			expected:  50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := durationToScore(tt.elapsed, tt.bestTime, tt.worstTime)
			if score < 0 || score > 100 {
				t.Errorf("score %d out of range [0, 100]", score)
			}
			// Allow ±5 tolerance for midpoint test
			tolerance := 5
			if score < tt.expected-tolerance || score > tt.expected+tolerance {
				t.Errorf("durationToScore = %d, want ~%d (±%d)", score, tt.expected, tolerance)
			}
		})
	}
}

func TestDurationToScoreAlwaysInRange(t *testing.T) {
	durations := []time.Duration{
		0,
		1 * time.Nanosecond,
		1 * time.Millisecond,
		100 * time.Millisecond,
		1 * time.Second,
		10 * time.Second,
		1 * time.Minute,
	}

	for _, d := range durations {
		score := durationToScore(d, 5*time.Millisecond, 500*time.Millisecond)
		if score < 0 || score > 100 {
			t.Errorf("durationToScore(%v) = %d, out of range [0, 100]", d, score)
		}
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

func TestGetDiskUsageAverage(t *testing.T) {
	avg := getDiskUsageAverage()
	if avg < 0 || avg > 100 {
		t.Errorf("disk usage average out of range [0, 100]: %f", avg)
	}
	t.Logf("Disk usage average: %.2f%%", avg)
}

func TestRunBenchmark(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping benchmark test in short mode")
	}

	result, err := RunBenchmark()
	if err != nil {
		t.Fatalf("RunBenchmark returned error: %v", err)
	}

	if result == nil {
		t.Fatal("RunBenchmark returned nil")
	}

	if result.CPUScore < 0 || result.CPUScore > 100 {
		t.Errorf("CPUScore %d out of range [0, 100]", result.CPUScore)
	}
	if result.RAMScore < 0 || result.RAMScore > 100 {
		t.Errorf("RAMScore %d out of range [0, 100]", result.RAMScore)
	}
	if result.DiskScore < 0 || result.DiskScore > 100 {
		t.Errorf("DiskScore %d out of range [0, 100]", result.DiskScore)
	}
	if result.OverallScore < 0 || result.OverallScore > 100 {
		t.Errorf("OverallScore %d out of range [0, 100]", result.OverallScore)
	}
	if result.Duration == "" {
		t.Error("Duration is empty")
	}

	t.Logf("Benchmark: cpu=%d, ram=%d, disk=%d, overall=%d, duration=%s",
		result.CPUScore, result.RAMScore, result.DiskScore, result.OverallScore, result.Duration)
}

func TestBenchmarkScoresInRange(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping benchmark score test in short mode")
	}

	result, err := RunBenchmark()
	if err != nil {
		t.Fatalf("RunBenchmark returned error: %v", err)
	}

	scores := map[string]int{
		"CPUScore":     result.CPUScore,
		"RAMScore":     result.RAMScore,
		"DiskScore":    result.DiskScore,
		"OverallScore": result.OverallScore,
	}

	for name, score := range scores {
		t.Run(name, func(t *testing.T) {
			if score < 0 {
				t.Errorf("%s is negative: %d", name, score)
			}
			if score > 100 {
				t.Errorf("%s exceeds 100: %d", name, score)
			}
		})
	}
}

func TestCheckThermalThrottling(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("CheckThermalThrottling panicked: %v", r)
		}
	}()

	alerts, err := CheckThermalThrottling()
	if err != nil {
		t.Fatalf("CheckThermalThrottling returned error: %v", err)
	}

	for _, alert := range alerts {
		if alert.Component == "" {
			t.Error("alert has empty Component")
		}
		if alert.Message == "" {
			t.Error("alert has empty Message")
		}
		t.Logf("Thermal alert: %s at %.1f°C (threshold: %.0f°C)", alert.Component, alert.Temp, alert.Threshold)
	}
}
