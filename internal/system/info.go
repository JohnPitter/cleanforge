package system

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	"cleanforge/internal/cmd"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
)

// staticCache holds hardware info that never changes during a session.
var (
	staticOnce  sync.Once
	staticCache *staticInfo
)

type staticInfo struct {
	OS         string
	Hostname   string
	Platform   string
	CPUModel   string
	CPUCores   int
	CPUThreads int
	RAMTotal   uint64
	RAMModules []RAMModule
	GPUName    string
	GPUDriver  string
	GPUs       []GPUDetail
	PhysDisks  []PhysDisk
}

// loadStaticInfo collects hardware details that don't change at runtime.
// Called once and cached via sync.Once.
func loadStaticInfo() *staticInfo {
	staticOnce.Do(func() {
		s := &staticInfo{}

		// OS info
		s.OS = runtime.GOOS
		hostInfo, err := host.Info()
		if err == nil {
			s.Hostname = hostInfo.Hostname
			s.Platform = hostInfo.Platform + " " + hostInfo.PlatformVersion
		}

		// CPU info
		cpuInfos, err := cpu.Info()
		if err == nil && len(cpuInfos) > 0 {
			s.CPUModel = cpuInfos[0].ModelName
		}
		physicalCores, err := cpu.Counts(false)
		if err == nil {
			s.CPUCores = physicalCores
		}
		logicalCores, err := cpu.Counts(true)
		if err == nil {
			s.CPUThreads = logicalCores
		}

		// RAM total
		ramInfo, _ := mem.VirtualMemory()
		if ramInfo != nil {
			s.RAMTotal = ramInfo.Total
		}

		// Hardware details (WMI / PowerShell — slow, only once)
		s.RAMModules = GetRAMModules()
		s.GPUName, s.GPUDriver = GetGPUInfo()
		s.GPUs = GetGPUDetails()
		s.PhysDisks = GetPhysicalDisks()

		staticCache = s
	})
	return staticCache
}

// SystemInfo holds comprehensive system information for the Dashboard.
type SystemInfo struct {
	OS          string     `json:"os"`
	Hostname    string     `json:"hostname"`
	Platform    string     `json:"platform"`
	CPUModel    string     `json:"cpuModel"`
	CPUCores    int        `json:"cpuCores"`
	CPUThreads  int        `json:"cpuThreads"`
	CPUUsage    float64    `json:"cpuUsage"`
	RAMTotal    uint64     `json:"ramTotal"`
	RAMUsed     uint64     `json:"ramUsed"`
	RAMUsage    float64    `json:"ramUsage"`
	RAMModules  []RAMModule `json:"ramModules"`
	GPUName     string     `json:"gpuName"`
	GPUDriver   string     `json:"gpuDriver"`
	GPUs        []GPUDetail `json:"gpus"`
	Disks       []DiskInfo `json:"disks"`
	PhysDisks   []PhysDisk `json:"physDisks"`
	Uptime      string     `json:"uptime"`
	HealthScore int        `json:"healthScore"`
}

// RAMModule represents a single physical memory stick.
type RAMModule struct {
	Manufacturer string `json:"manufacturer"`
	Capacity     uint64 `json:"capacity"`
	Speed        uint32 `json:"speed"`
	PartNumber   string `json:"partNumber"`
	FormFactor   string `json:"formFactor"`
	Slot         string `json:"slot"`
}

// GPUDetail represents a single GPU adapter.
type GPUDetail struct {
	Name    string `json:"name"`
	Driver  string `json:"driver"`
	VRAM    uint64 `json:"vram"`
}

// PhysDisk represents a physical disk drive.
type PhysDisk struct {
	Model     string `json:"model"`
	Size      uint64 `json:"size"`
	MediaType string `json:"mediaType"`
	Interface string `json:"interface"`
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
// Static hardware data (CPU model, RAM modules, GPU, physical disks) is cached after the first call.
// Only dynamic metrics (CPU/RAM usage, disk usage, uptime, health) are refreshed each call.
func GetSystemInfo() (*SystemInfo, error) {
	s := loadStaticInfo()

	info := &SystemInfo{
		// Cached static data
		OS:         s.OS,
		Hostname:   s.Hostname,
		Platform:   s.Platform,
		CPUModel:   s.CPUModel,
		CPUCores:   s.CPUCores,
		CPUThreads: s.CPUThreads,
		RAMTotal:   s.RAMTotal,
		RAMModules: s.RAMModules,
		GPUName:    s.GPUName,
		GPUDriver:  s.GPUDriver,
		GPUs:       s.GPUs,
		PhysDisks:  s.PhysDisks,
	}

	// Dynamic data — refreshed every call
	cpuUsage, err := GetCPUUsage()
	if err == nil {
		info.CPUUsage = cpuUsage
	}

	ramInfo, err := GetRAMUsage()
	if err == nil {
		info.RAMUsed = ramInfo.Used
		info.RAMUsage = ramInfo.UsedPercent
	}

	disks, err := GetDiskUsage()
	if err == nil {
		info.Disks = disks
	}

	uptimeSecs, err := host.Uptime()
	if err == nil {
		info.Uptime = formatUptime(uptimeSecs)
	}

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

// detectRAMManufacturer infers the manufacturer from the part number prefix
// when WMI returns "Unknown" or empty.
func detectRAMManufacturer(partNumber, wmiManufacturer string) string {
	mfr := strings.TrimSpace(wmiManufacturer)
	if mfr != "" && !strings.EqualFold(mfr, "unknown") {
		return mfr
	}

	pn := strings.ToUpper(strings.TrimSpace(partNumber))
	if pn == "" {
		return mfr
	}

	// Map part-number prefixes to manufacturers
	prefixes := []struct {
		prefix string
		name   string
	}{
		{"CMW", "Corsair"},
		{"CMK", "Corsair"},
		{"CMR", "Corsair"},
		{"CML", "Corsair"},
		{"CMH", "Corsair"},
		{"CMT", "Corsair"},
		{"CM", "Corsair"},
		{"KVR", "Kingston"},
		{"KHX", "Kingston"},
		{"FURY", "Kingston"},
		{"HX", "Kingston"},
		{"F5-", "G.Skill"},
		{"F4-", "G.Skill"},
		{"F3-", "G.Skill"},
		{"BL", "Crucial"},
		{"CT", "Crucial"},
		{"BLS", "Crucial"},
		{"HMA", "SK Hynix"},
		{"HMT", "SK Hynix"},
		{"HMCG", "SK Hynix"},
		{"HMAA", "SK Hynix"},
		{"M378", "Samsung"},
		{"M471", "Samsung"},
		{"M393", "Samsung"},
		{"M3", "Samsung"},
		{"PVS", "Patriot"},
		{"PV", "Patriot"},
		{"AD", "ADATA"},
		{"AX", "ADATA"},
		{"TF", "Team Group"},
		{"TD", "Team Group"},
		{"TLZGD", "Team Group"},
	}

	for _, p := range prefixes {
		if strings.HasPrefix(pn, p.prefix) {
			return p.name
		}
	}

	return mfr
}

// GetRAMModules queries individual RAM sticks via WMI.
func GetRAMModules() []RAMModule {
	if runtime.GOOS != "windows" {
		return nil
	}

	out, err := cmd.Hidden("wmic", "memorychip", "get", "Manufacturer,Capacity,Speed,PartNumber,DeviceLocator,FormFactor,BankLabel", "/format:csv").Output()
	if err != nil {
		return nil
	}

	var modules []RAMModule
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for idx, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Node") {
			continue
		}
		// CSV: Node,BankLabel,Capacity,DeviceLocator,FormFactor,Manufacturer,PartNumber,Speed
		parts := strings.Split(line, ",")
		if len(parts) < 8 {
			continue
		}
		var capacity uint64
		fmt.Sscan(strings.TrimSpace(parts[2]), &capacity)
		var speed uint32
		fmt.Sscan(strings.TrimSpace(parts[7]), &speed)
		var formFactorNum int
		fmt.Sscan(strings.TrimSpace(parts[4]), &formFactorNum)

		ff := "Unknown"
		switch formFactorNum {
		case 8:
			ff = "DIMM"
		case 12:
			ff = "SO-DIMM"
		}

		partNumber := strings.TrimSpace(parts[6])
		manufacturer := detectRAMManufacturer(partNumber, parts[5])

		// Build slot label: prefer BankLabel/DeviceLocator, fall back to index
		slot := strings.TrimSpace(parts[1])
		devLocator := strings.TrimSpace(parts[3])
		if devLocator != "" && devLocator != slot {
			slot = devLocator
		}
		if slot == "" {
			slot = fmt.Sprintf("Slot %d", idx)
		}

		modules = append(modules, RAMModule{
			Manufacturer: manufacturer,
			Capacity:     capacity,
			Speed:        speed,
			PartNumber:   partNumber,
			Slot:         slot,
			FormFactor:   ff,
		})
	}
	return modules
}

// GetGPUDetails queries all GPU adapters with VRAM info.
// Uses PowerShell to read qwMemorySize which supports >4 GB (AdapterRAM is 32-bit and overflows).
func GetGPUDetails() []GPUDetail {
	if runtime.GOOS != "windows" {
		return nil
	}

	// PowerShell query returns correct 64-bit VRAM via registry qwMemorySize
	psScript := `Get-CimInstance Win32_VideoController | ForEach-Object {
		$vram = 0
		$regPath = "HKLM:\SYSTEM\ControlSet001\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}"
		$subkeys = Get-ChildItem $regPath -ErrorAction SilentlyContinue | Where-Object { $_.Name -match '\\\d{4}$' }
		foreach ($sk in $subkeys) {
			$desc = (Get-ItemProperty $sk.PSPath -ErrorAction SilentlyContinue).'DriverDesc'
			if ($desc -eq $_.Name) {
				$qw = (Get-ItemProperty $sk.PSPath -ErrorAction SilentlyContinue).'HardwareInformation.qwMemorySize'
				if ($qw) { $vram = $qw; break }
			}
		}
		if ($vram -eq 0) { $vram = $_.AdapterRAM }
		"$($_.Name)|$($_.DriverVersion)|$vram"
	}`

	out, err := cmd.Hidden("powershell", "-NoProfile", "-Command", psScript).Output()
	if err != nil {
		return getGPUDetailsFallback()
	}

	var gpus []GPUDetail
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 3 {
			continue
		}
		var vram uint64
		fmt.Sscan(strings.TrimSpace(parts[2]), &vram)

		gpus = append(gpus, GPUDetail{
			Name:   strings.TrimSpace(parts[0]),
			Driver: strings.TrimSpace(parts[1]),
			VRAM:   vram,
		})
	}
	if len(gpus) > 0 {
		return gpus
	}
	return getGPUDetailsFallback()
}

// getGPUDetailsFallback uses wmic (AdapterRAM is 32-bit, caps at ~4 GB).
func getGPUDetailsFallback() []GPUDetail {
	out, err := cmd.Hidden("wmic", "path", "win32_VideoController", "get", "Name,DriverVersion,AdapterRAM", "/format:csv").Output()
	if err != nil {
		return nil
	}

	var gpus []GPUDetail
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Node") {
			continue
		}
		// CSV: Node,AdapterRAM,DriverVersion,Name
		parts := strings.Split(line, ",")
		if len(parts) < 4 {
			continue
		}
		var vram uint64
		fmt.Sscan(strings.TrimSpace(parts[1]), &vram)

		gpus = append(gpus, GPUDetail{
			Name:   strings.TrimSpace(parts[3]),
			Driver: strings.TrimSpace(parts[2]),
			VRAM:   vram,
		})
	}
	return gpus
}

// GetPhysicalDisks queries physical disk drives via WMI (model, size, type).
func GetPhysicalDisks() []PhysDisk {
	if runtime.GOOS != "windows" {
		return nil
	}

	out, err := cmd.Hidden("wmic", "diskdrive", "get", "Model,Size,MediaType,InterfaceType", "/format:csv").Output()
	if err != nil {
		return nil
	}

	var disks []PhysDisk
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Node") {
			continue
		}
		// CSV: Node,InterfaceType,MediaType,Model,Size
		parts := strings.Split(line, ",")
		if len(parts) < 5 {
			continue
		}
		var size uint64
		fmt.Sscan(strings.TrimSpace(parts[4]), &size)

		mediaType := strings.TrimSpace(parts[2])
		if mediaType == "" {
			mediaType = "SSD"
		}

		disks = append(disks, PhysDisk{
			Model:     strings.TrimSpace(parts[3]),
			Size:      size,
			MediaType: mediaType,
			Interface: strings.TrimSpace(parts[1]),
		})
	}
	return disks
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
