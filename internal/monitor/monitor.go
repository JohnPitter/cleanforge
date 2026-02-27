package monitor

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"cleanforge/internal/cmd"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
)

// MonitorSnapshot holds a point-in-time snapshot of system metrics.
type MonitorSnapshot struct {
	Timestamp int64   `json:"timestamp"`
	CPUUsage  float64 `json:"cpuUsage"`
	RAMUsage  float64 `json:"ramUsage"`
	GPUTemp   float64 `json:"gpuTemp"`
	CPUTemp   float64 `json:"cpuTemp"`
	DiskUsage float64 `json:"diskUsage"`
	FanSpeed  int     `json:"fanSpeed"`
}

// BenchmarkResult holds the results of a simple system benchmark.
type BenchmarkResult struct {
	CPUScore     int    `json:"cpuScore"`
	RAMScore     int    `json:"ramScore"`
	DiskScore    int    `json:"diskScore"`
	OverallScore int    `json:"overallScore"`
	Duration     string `json:"duration"`
}

// ThermalAlert represents a thermal throttling warning for a component.
type ThermalAlert struct {
	Component string  `json:"component"`
	Temp      float64 `json:"temp"`
	Threshold float64 `json:"threshold"`
	Message   string  `json:"message"`
}

// GetSnapshot gathers all system metrics at once and returns a MonitorSnapshot.
func GetSnapshot() (*MonitorSnapshot, error) {
	snapshot := &MonitorSnapshot{
		Timestamp: time.Now().Unix(),
	}

	// CPU usage (sampled over 1 second)
	cpuPercents, err := cpu.Percent(1*time.Second, false)
	if err == nil && len(cpuPercents) > 0 {
		snapshot.CPUUsage = roundFloat(cpuPercents[0], 2)
	}

	// RAM usage
	vmStat, err := mem.VirtualMemory()
	if err == nil {
		snapshot.RAMUsage = roundFloat(vmStat.UsedPercent, 2)
	}

	// Disk usage (average across all partitions)
	snapshot.DiskUsage = getDiskUsageAverage()

	// CPU temperature
	cpuTemp, err := GetCPUTemp()
	if err == nil {
		snapshot.CPUTemp = cpuTemp
	}

	// GPU temperature
	gpuTemp, err := GetGPUTemp()
	if err == nil {
		snapshot.GPUTemp = gpuTemp
	}

	// Fan speed (best-effort via WMI)
	snapshot.FanSpeed = getFanSpeed()

	return snapshot, nil
}

// GetCPUTemp attempts to read the CPU temperature in Celsius.
// It tries WMI MSAcpi_ThermalZoneTemperature first (requires admin on some systems),
// then falls back to Open Hardware Monitor WMI if available.
func GetCPUTemp() (float64, error) {
	// Method 1: MSAcpi_ThermalZoneTemperature via WMI
	// This returns temperature in tenths of Kelvin
	out, err := cmd.Hidden("powershell", "-NoProfile", "-Command",
		"Get-CimInstance MSAcpi_ThermalZoneTemperature -Namespace root/WMI -ErrorAction SilentlyContinue | Select-Object -ExpandProperty CurrentTemperature -First 1",
	).Output()
	if err == nil {
		tempStr := strings.TrimSpace(string(out))
		if tempStr != "" {
			tempVal, parseErr := strconv.ParseFloat(tempStr, 64)
			if parseErr == nil && tempVal > 0 {
				// Convert from tenths of Kelvin to Celsius
				celsius := (tempVal / 10.0) - 273.15
				if celsius > 0 && celsius < 150 {
					return roundFloat(celsius, 1), nil
				}
			}
		}
	}

	// Method 2: Open Hardware Monitor / LibreHardwareMonitor WMI
	out, err = cmd.Hidden("powershell", "-NoProfile", "-Command",
		`Get-CimInstance -Namespace root/OpenHardwareMonitor -ClassName Sensor -ErrorAction SilentlyContinue | Where-Object { $_.SensorType -eq 'Temperature' -and $_.Name -like '*CPU*' } | Select-Object -ExpandProperty Value -First 1`,
	).Output()
	if err == nil {
		tempStr := strings.TrimSpace(string(out))
		if tempStr != "" {
			tempVal, parseErr := strconv.ParseFloat(tempStr, 64)
			if parseErr == nil && tempVal > 0 && tempVal < 150 {
				return roundFloat(tempVal, 1), nil
			}
		}
	}

	// Method 3: LibreHardwareMonitor WMI namespace
	out, err = cmd.Hidden("powershell", "-NoProfile", "-Command",
		`Get-CimInstance -Namespace root/LibreHardwareMonitor -ClassName Sensor -ErrorAction SilentlyContinue | Where-Object { $_.SensorType -eq 'Temperature' -and $_.Name -like '*CPU*' } | Select-Object -ExpandProperty Value -First 1`,
	).Output()
	if err == nil {
		tempStr := strings.TrimSpace(string(out))
		if tempStr != "" {
			tempVal, parseErr := strconv.ParseFloat(tempStr, 64)
			if parseErr == nil && tempVal > 0 && tempVal < 150 {
				return roundFloat(tempVal, 1), nil
			}
		}
	}

	return 0, fmt.Errorf("could not read CPU temperature: no supported method available")
}

// GetGPUTemp attempts to read the GPU temperature in Celsius.
// It tries nvidia-smi for NVIDIA GPUs first, then falls back to WMI queries
// for AMD or other vendors.
func GetGPUTemp() (float64, error) {
	// Method 1: NVIDIA GPU via nvidia-smi
	out, err := cmd.Hidden("nvidia-smi",
		"--query-gpu=temperature.gpu",
		"--format=csv,noheader,nounits",
	).Output()
	if err == nil {
		tempStr := strings.TrimSpace(string(out))
		// nvidia-smi may return multiple lines for multi-GPU; take the first
		lines := strings.Split(tempStr, "\n")
		if len(lines) > 0 {
			tempVal, parseErr := strconv.ParseFloat(strings.TrimSpace(lines[0]), 64)
			if parseErr == nil && tempVal > 0 && tempVal < 150 {
				return roundFloat(tempVal, 1), nil
			}
		}
	}

	// Method 2: Open Hardware Monitor WMI for GPU temperature
	out, err = cmd.Hidden("powershell", "-NoProfile", "-Command",
		`Get-CimInstance -Namespace root/OpenHardwareMonitor -ClassName Sensor -ErrorAction SilentlyContinue | Where-Object { $_.SensorType -eq 'Temperature' -and $_.Name -like '*GPU*' } | Select-Object -ExpandProperty Value -First 1`,
	).Output()
	if err == nil {
		tempStr := strings.TrimSpace(string(out))
		if tempStr != "" {
			tempVal, parseErr := strconv.ParseFloat(tempStr, 64)
			if parseErr == nil && tempVal > 0 && tempVal < 150 {
				return roundFloat(tempVal, 1), nil
			}
		}
	}

	// Method 3: LibreHardwareMonitor WMI namespace for GPU
	out, err = cmd.Hidden("powershell", "-NoProfile", "-Command",
		`Get-CimInstance -Namespace root/LibreHardwareMonitor -ClassName Sensor -ErrorAction SilentlyContinue | Where-Object { $_.SensorType -eq 'Temperature' -and $_.Name -like '*GPU*' } | Select-Object -ExpandProperty Value -First 1`,
	).Output()
	if err == nil {
		tempStr := strings.TrimSpace(string(out))
		if tempStr != "" {
			tempVal, parseErr := strconv.ParseFloat(tempStr, 64)
			if parseErr == nil && tempVal > 0 && tempVal < 150 {
				return roundFloat(tempVal, 1), nil
			}
		}
	}

	return 0, fmt.Errorf("could not read GPU temperature: no supported method available")
}

// RunBenchmark executes a simple system benchmark that tests CPU, RAM, and disk
// performance. Each component is scored from 0 to 100, and an overall weighted
// score is calculated.
func RunBenchmark() (*BenchmarkResult, error) {
	totalStart := time.Now()

	// CPU Benchmark: compute prime numbers up to 1,000,000
	cpuScore := benchmarkCPU()

	// RAM Benchmark: allocate, write, and read 100MB
	ramScore := benchmarkRAM()

	// Disk Benchmark: write and read a 100MB temp file
	diskScore := benchmarkDisk()

	totalDuration := time.Since(totalStart)

	// Overall score: weighted average (CPU 40%, RAM 30%, Disk 30%)
	overallScore := int(float64(cpuScore)*0.4 + float64(ramScore)*0.3 + float64(diskScore)*0.3)
	if overallScore > 100 {
		overallScore = 100
	}
	if overallScore < 0 {
		overallScore = 0
	}

	return &BenchmarkResult{
		CPUScore:     cpuScore,
		RAMScore:     ramScore,
		DiskScore:    diskScore,
		OverallScore: overallScore,
		Duration:     totalDuration.Round(time.Millisecond).String(),
	}, nil
}

// CheckThermalThrottling checks CPU and GPU temperatures against thresholds
// and returns alerts for any components that are too hot.
func CheckThermalThrottling() ([]ThermalAlert, error) {
	const cpuThreshold = 85.0
	const gpuThreshold = 80.0

	var alerts []ThermalAlert

	cpuTemp, err := GetCPUTemp()
	if err == nil && cpuTemp >= cpuThreshold {
		alerts = append(alerts, ThermalAlert{
			Component: "CPU",
			Temp:      cpuTemp,
			Threshold: cpuThreshold,
			Message:   fmt.Sprintf("CPU temperature is %.1f°C, exceeding the %.0f°C threshold. Thermal throttling may occur.", cpuTemp, cpuThreshold),
		})
	}

	gpuTemp, err := GetGPUTemp()
	if err == nil && gpuTemp >= gpuThreshold {
		alerts = append(alerts, ThermalAlert{
			Component: "GPU",
			Temp:      gpuTemp,
			Threshold: gpuThreshold,
			Message:   fmt.Sprintf("GPU temperature is %.1f°C, exceeding the %.0f°C threshold. Thermal throttling may occur.", gpuTemp, gpuThreshold),
		})
	}

	return alerts, nil
}

// benchmarkCPU measures how long it takes to find all primes up to 1,000,000
// using a sieve, then scores 0-100 based on duration.
func benchmarkCPU() int {
	start := time.Now()

	// Sieve of Eratosthenes up to 1,000,000
	const limit = 1_000_000
	sieve := make([]bool, limit+1)
	for i := 2; i <= limit; i++ {
		sieve[i] = true
	}
	for i := 2; i*i <= limit; i++ {
		if sieve[i] {
			for j := i * i; j <= limit; j += i {
				sieve[j] = false
			}
		}
	}

	// Count primes to prevent the compiler from optimizing the loop away
	count := 0
	for i := 2; i <= limit; i++ {
		if sieve[i] {
			count++
		}
	}
	_ = count

	elapsed := time.Since(start)
	return durationToScore(elapsed, 5*time.Millisecond, 500*time.Millisecond)
}

// benchmarkRAM measures how long it takes to allocate, write, and read 100MB of data.
func benchmarkRAM() int {
	const size = 100 * 1024 * 1024 // 100 MB

	start := time.Now()

	// Allocate and write
	buf := make([]byte, size)
	for i := range buf {
		buf[i] = byte(i % 256)
	}

	// Read and verify (prevent optimization)
	checksum := uint64(0)
	for _, b := range buf {
		checksum += uint64(b)
	}
	_ = checksum

	elapsed := time.Since(start)
	return durationToScore(elapsed, 30*time.Millisecond, 2*time.Second)
}

// benchmarkDisk measures how long it takes to write and read a 100MB temp file.
func benchmarkDisk() int {
	const size = 100 * 1024 * 1024 // 100 MB

	// Create temp directory if needed
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, "cleanforge_benchmark.tmp")

	// Clean up on exit
	defer os.Remove(tmpFile)

	// Generate random data to prevent filesystem compression shortcuts
	data := make([]byte, size)
	_, err := rand.Read(data)
	if err != nil {
		// Fallback to pseudo-random pattern
		for i := range data {
			data[i] = byte((i * 37 + 17) % 256)
		}
	}

	start := time.Now()

	// Write phase
	err = os.WriteFile(tmpFile, data, 0644)
	if err != nil {
		return 0
	}

	// Read phase
	readData, err := os.ReadFile(tmpFile)
	if err != nil {
		return 10 // Partial score: write succeeded
	}

	// Verify to prevent optimization
	_ = len(readData)

	elapsed := time.Since(start)
	return durationToScore(elapsed, 100*time.Millisecond, 10*time.Second)
}

// durationToScore converts a benchmark duration to a 0-100 score.
// bestTime gets 100, worstTime gets 0, linear interpolation in between.
func durationToScore(elapsed, bestTime, worstTime time.Duration) int {
	if elapsed <= bestTime {
		return 100
	}
	if elapsed >= worstTime {
		return 0
	}

	// Linear interpolation
	totalRange := float64(worstTime - bestTime)
	position := float64(elapsed - bestTime)
	score := int(100.0 * (1.0 - position/totalRange))

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return score
}

// getDiskUsageAverage returns the average disk usage percentage across all partitions.
func getDiskUsageAverage() float64 {
	partitions, err := disk.Partitions(false)
	if err != nil || len(partitions) == 0 {
		return 0
	}

	var totalUsage float64
	var count int
	for _, p := range partitions {
		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			continue
		}
		totalUsage += usage.UsedPercent
		count++
	}

	if count == 0 {
		return 0
	}
	return roundFloat(totalUsage/float64(count), 2)
}

// getFanSpeed attempts to read fan speed via WMI. Returns 0 if unavailable.
func getFanSpeed() int {
	// Try Win32_Fan WMI class
	out, err := cmd.Hidden("powershell", "-NoProfile", "-Command",
		"Get-CimInstance Win32_Fan -ErrorAction SilentlyContinue | Select-Object -ExpandProperty DesiredSpeed -First 1",
	).Output()
	if err == nil {
		speedStr := strings.TrimSpace(string(out))
		if speedStr != "" {
			speed, parseErr := strconv.Atoi(speedStr)
			if parseErr == nil && speed > 0 {
				return speed
			}
		}
	}

	// Try Open Hardware Monitor
	out, err = cmd.Hidden("powershell", "-NoProfile", "-Command",
		`Get-CimInstance -Namespace root/OpenHardwareMonitor -ClassName Sensor -ErrorAction SilentlyContinue | Where-Object { $_.SensorType -eq 'Fan' } | Select-Object -ExpandProperty Value -First 1`,
	).Output()
	if err == nil {
		speedStr := strings.TrimSpace(string(out))
		if speedStr != "" {
			speedFloat, parseErr := strconv.ParseFloat(speedStr, 64)
			if parseErr == nil && speedFloat > 0 {
				return int(speedFloat)
			}
		}
	}

	return 0
}

// roundFloat rounds a float64 to the specified number of decimal places.
func roundFloat(val float64, places int) float64 {
	factor := 1.0
	for i := 0; i < places; i++ {
		factor *= 10
	}
	return float64(int(val*factor+0.5)) / factor
}

