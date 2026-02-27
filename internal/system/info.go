package system

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"cleanforge/internal/cmd"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
)

// SystemInfo holds comprehensive system information for the Dashboard.
type SystemInfo struct {
	OS         string  `json:"os"`
	Hostname   string  `json:"hostname"`
	Platform   string  `json:"platform"`
	CPUModel   string  `json:"cpuModel"`
	CPUCores   int     `json:"cpuCores"`
	CPUThreads int     `json:"cpuThreads"`
	CPUUsage   float64 `json:"cpuUsage"`
	RAMTotal   uint64  `json:"ramTotal"`
	RAMUsed    uint64  `json:"ramUsed"`
	RAMUsage   float64 `json:"ramUsage"`
	GPUName    string  `json:"gpuName"`
	GPUDriver  string  `json:"gpuDriver"`
	Disks      []DiskInfo `json:"disks"`
	Uptime     string  `json:"uptime"`
	HealthScore int    `json:"healthScore"`
}

// DiskInfo holds usage information for a single disk partition.
type DiskInfo struct {
	Drive        string  `json:"drive"`
	Total        uint64  `json:"total"`
	Used         uint64  `json:"used"`
	Free         uint64  `json:"free"`
	UsagePercent float64 `json:"usagePercent"`
	FSType       string  `json:"fsType"`
}

// GetSystemInfo gathers all system information including CPU, RAM, GPU, disks, and uptime.
func GetSystemInfo() (*SystemInfo, error) {
	info := &SystemInfo{}

	// OS info
	info.OS = runtime.GOOS
	hostInfo, err := host.Info()
	if err == nil {
		info.Hostname = hostInfo.Hostname
		info.Platform = hostInfo.Platform + " " + hostInfo.PlatformVersion
	}

	// CPU info
	cpuInfos, err := cpu.Info()
	if err == nil && len(cpuInfos) > 0 {
		info.CPUModel = cpuInfos[0].ModelName
	}

	physicalCores, err := cpu.Counts(false)
	if err == nil {
		info.CPUCores = physicalCores
	}

	logicalCores, err := cpu.Counts(true)
	if err == nil {
		info.CPUThreads = logicalCores
	}

	// CPU usage (sampled over 1 second)
	cpuUsage, err := GetCPUUsage()
	if err == nil {
		info.CPUUsage = cpuUsage
	}

	// RAM info
	ramInfo, err := GetRAMUsage()
	if err == nil {
		info.RAMTotal = ramInfo.Total
		info.RAMUsed = ramInfo.Used
		info.RAMUsage = ramInfo.UsedPercent
	}

	// GPU info
	info.GPUName, info.GPUDriver = GetGPUInfo()

	// Disk info
	disks, err := GetDiskUsage()
	if err == nil {
		info.Disks = disks
	}

	// Uptime
	uptimeSecs, err := host.Uptime()
	if err == nil {
		info.Uptime = formatUptime(uptimeSecs)
	}

	// Health score
	info.HealthScore = CalculateHealthScore(info)

	return info, nil
}

// GetCPUUsage returns the current aggregate CPU usage percentage sampled over 1 second.
func GetCPUUsage() (float64, error) {
	percentages, err := cpu.Percent(1*time.Second, false)
	if err != nil {
		return 0, fmt.Errorf("failed to get CPU usage: %w", err)
	}
	if len(percentages) == 0 {
		return 0, fmt.Errorf("no CPU usage data returned")
	}
	return math_round(percentages[0], 2), nil
}

// GetRAMUsage returns virtual memory statistics.
func GetRAMUsage() (*mem.VirtualMemoryStat, error) {
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get RAM usage: %w", err)
	}
	return vmStat, nil
}

// GetDiskUsage returns usage information for all mounted disk partitions.
func GetDiskUsage() ([]DiskInfo, error) {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return nil, fmt.Errorf("failed to get disk partitions: %w", err)
	}

	var disks []DiskInfo
	for _, partition := range partitions {
		usage, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			continue
		}

		disks = append(disks, DiskInfo{
			Drive:        partition.Mountpoint,
			Total:        usage.Total,
			Used:         usage.Used,
			Free:         usage.Free,
			UsagePercent: math_round(usage.UsedPercent, 2),
			FSType:       partition.Fstype,
		})
	}

	return disks, nil
}

// GetGPUInfo detects the GPU name and driver version using wmic on Windows.
// Returns empty strings on non-Windows platforms or if detection fails.
func GetGPUInfo() (name string, driver string) {
	if runtime.GOOS != "windows" {
		return "", ""
	}

	out, err := cmd.Hidden("wmic", "path", "win32_VideoController", "get", "Name,DriverVersion", "/format:csv").Output()
	if err != nil {
		return "", ""
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	// CSV output format: Node,DriverVersion,Name
	// First non-empty line after header contains the data
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Node") {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) >= 3 {
			driver = strings.TrimSpace(parts[1])
			name = strings.TrimSpace(parts[2])
			// Prefer discrete GPU (NVIDIA or AMD) over integrated
			upperName := strings.ToUpper(name)
			if strings.Contains(upperName, "NVIDIA") || strings.Contains(upperName, "AMD") || strings.Contains(upperName, "RADEON") {
				return name, driver
			}
		}
	}

	// If no discrete GPU found, return the last parsed values (likely integrated)
	return name, driver
}

// CalculateHealthScore computes a system health score from 0 to 100 based on
// CPU usage, RAM usage, disk free space, and system uptime.
func CalculateHealthScore(info *SystemInfo) int {
	score := 100

	// CPU usage penalty: <50% = good (no penalty), 50-80% = moderate, >80% = bad
	if info.CPUUsage >= 80 {
		score -= 30
	} else if info.CPUUsage >= 50 {
		score -= 15
	}

	// RAM usage penalty: <70% = good, 70-90% = moderate, >90% = bad
	if info.RAMUsage >= 90 {
		score -= 30
	} else if info.RAMUsage >= 70 {
		score -= 15
	}

	// Disk space penalty: check each disk; penalize if any disk has <20% free
	worstDiskPenalty := 0
	for _, d := range info.Disks {
		freePercent := 100.0 - d.UsagePercent
		if freePercent < 10 {
			if worstDiskPenalty < 25 {
				worstDiskPenalty = 25
			}
		} else if freePercent < 20 {
			if worstDiskPenalty < 15 {
				worstDiskPenalty = 15
			}
		}
	}
	score -= worstDiskPenalty

	// Uptime penalty: <7 days = good, 7-14 days = moderate, >14 days = bad
	uptimeSecs, err := host.Uptime()
	if err == nil {
		uptimeDays := uptimeSecs / 86400
		if uptimeDays > 14 {
			score -= 15
		} else if uptimeDays >= 7 {
			score -= 5
		}
	}

	if score < 0 {
		score = 0
	}

	return score
}

// formatUptime converts seconds into a human-readable string like "3d 5h 23m".
func formatUptime(seconds uint64) string {
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	minutes := (seconds % 3600) / 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

// math_round rounds a float64 to a specified number of decimal places.
func math_round(val float64, places int) float64 {
	factor := 1.0
	for i := 0; i < places; i++ {
		factor *= 10
	}
	return float64(int(val*factor+0.5)) / factor
}
