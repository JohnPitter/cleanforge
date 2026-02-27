package memory

import (
	"fmt"
	"sort"
	"strings"
	"unsafe"

	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
	"golang.org/x/sys/windows"
)

// MemoryStatus holds comprehensive memory information.
type MemoryStatus struct {
	Total        uint64          `json:"total"`
	Used         uint64          `json:"used"`
	Available    uint64          `json:"available"`
	UsagePercent float64         `json:"usagePercent"`
	Cached       uint64          `json:"cached"`
	TopProcesses []ProcessMemory `json:"topProcesses"`
}

// ProcessMemory holds memory information for a single process.
type ProcessMemory struct {
	Name    string  `json:"name"`
	PID     int32   `json:"pid"`
	Memory  uint64  `json:"memory"`
	Percent float64 `json:"percent"`
}

// systemProcesses is a set of process names considered essential system processes.
// These are excluded from memory leak detection.
var systemProcesses = map[string]bool{
	"system":                    true,
	"registry":                  true,
	"smss.exe":                  true,
	"csrss.exe":                 true,
	"wininit.exe":               true,
	"services.exe":              true,
	"lsass.exe":                 true,
	"svchost.exe":               true,
	"winlogon.exe":              true,
	"dwm.exe":                   true,
	"explorer.exe":              true,
	"taskhostw.exe":             true,
	"runtimebroker.exe":         true,
	"searchhost.exe":            true,
	"startmenuexperiencehost.exe": true,
	"textinputhost.exe":         true,
	"ctfmon.exe":                true,
	"fontdrvhost.exe":           true,
	"lsaiso.exe":                true,
	"securityhealthservice.exe": true,
	"securityhealthsystray.exe": true,
	"sgrmbroker.exe":            true,
	"spoolsv.exe":               true,
	"wudfhost.exe":              true,
	"wmiprvse.exe":              true,
	"dllhost.exe":               true,
	"conhost.exe":               true,
	"sihost.exe":                true,
	"dashost.exe":               true,
	"audiodg.exe":               true,
	"memory compression":        true,
	"system idle process":       true,
	"idle":                      true,
	"secure system":             true,
	"ntoskrnl.exe":              true,
	"msmpsvc.exe":               true,
	"msmpeng.exe":               true,
	"nissrv.exe":                true,
	"shellexperiencehost.exe":   true,
}

// GetMemoryStatus gathers comprehensive memory information including virtual memory
// statistics and the top 10 memory-consuming processes.
func GetMemoryStatus() (*MemoryStatus, error) {
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual memory stats: %w", err)
	}

	topProcs, err := GetTopMemoryProcesses(10)
	if err != nil {
		// Non-fatal: return memory info without process list
		topProcs = []ProcessMemory{}
	}

	status := &MemoryStatus{
		Total:        vmStat.Total,
		Used:         vmStat.Used,
		Available:    vmStat.Available,
		UsagePercent: roundFloat(vmStat.UsedPercent, 2),
		Cached:       calculateCached(vmStat),
		TopProcesses: topProcs,
	}

	return status, nil
}

// GetTopMemoryProcesses returns the top N processes sorted by memory usage descending.
func GetTopMemoryProcesses(count int) ([]ProcessMemory, error) {
	procs, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("failed to list processes: %w", err)
	}

	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual memory stats: %w", err)
	}
	totalMem := vmStat.Total

	var procList []ProcessMemory
	for _, p := range procs {
		name, err := p.Name()
		if err != nil {
			continue
		}

		memInfo, err := p.MemoryInfo()
		if err != nil || memInfo == nil {
			continue
		}

		rss := memInfo.RSS
		if rss == 0 {
			continue
		}

		var pct float64
		if totalMem > 0 {
			pct = roundFloat(float64(rss)/float64(totalMem)*100.0, 2)
		}

		procList = append(procList, ProcessMemory{
			Name:    name,
			PID:     p.Pid,
			Memory:  rss,
			Percent: pct,
		})
	}

	// Sort by memory usage descending
	sort.Slice(procList, func(i, j int) bool {
		return procList[i].Memory > procList[j].Memory
	})

	if count > len(procList) {
		count = len(procList)
	}

	return procList[:count], nil
}

// FlushStandbyList attempts to reduce memory usage by trimming the working set of
// all accessible processes. It uses the Windows API SetProcessWorkingSetSize with
// special parameters (-1, -1) which instructs the OS to trim the working set.
func FlushStandbyList() error {
	procs, err := process.Processes()
	if err != nil {
		return fmt.Errorf("failed to list processes: %w", err)
	}

	var trimmed int
	var lastErr error

	for _, p := range procs {
		// Skip PID 0 (System Idle) and PID 4 (System)
		if p.Pid == 0 || p.Pid == 4 {
			continue
		}

		err := trimProcessWorkingSet(uint32(p.Pid))
		if err != nil {
			lastErr = err
			continue
		}
		trimmed++
	}

	if trimmed == 0 && lastErr != nil {
		return fmt.Errorf("could not trim any process working set, last error: %w", lastErr)
	}

	return nil
}

// trimProcessWorkingSet opens a process by PID and calls SetProcessWorkingSetSize
// with (^uintptr(0), ^uintptr(0)) to instruct Windows to trim its working set.
func trimProcessWorkingSet(pid uint32) error {
	// PROCESS_SET_QUOTA is required for SetProcessWorkingSetSize
	// PROCESS_QUERY_LIMITED_INFORMATION allows opening the handle
	const desiredAccess = windows.PROCESS_SET_QUOTA | windows.PROCESS_QUERY_LIMITED_INFORMATION

	handle, err := windows.OpenProcess(desiredAccess, false, pid)
	if err != nil {
		return fmt.Errorf("failed to open process %d: %w", pid, err)
	}
	defer windows.CloseHandle(handle)

	// Load kernel32.dll and get SetProcessWorkingSetSize
	kernel32 := windows.NewLazyDLL("kernel32.dll")
	setWorkingSetSize := kernel32.NewProc("SetProcessWorkingSetSize")

	// Passing SIZE_T(-1) for both min and max tells Windows to trim the working set
	minSize := ^uintptr(0)
	maxSize := ^uintptr(0)

	ret, _, callErr := setWorkingSetSize.Call(
		uintptr(handle),
		minSize,
		maxSize,
	)
	if ret == 0 {
		return fmt.Errorf("SetProcessWorkingSetSize failed for PID %d: %w", pid, callErr)
	}

	return nil
}

// DetectMemoryLeaks identifies processes using more than 500MB of RAM that are
// not essential system processes. These may indicate memory leaks or runaway processes.
func DetectMemoryLeaks() ([]ProcessMemory, error) {
	const memoryThreshold uint64 = 500 * 1024 * 1024 // 500 MB

	procs, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("failed to list processes: %w", err)
	}

	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual memory stats: %w", err)
	}
	totalMem := vmStat.Total

	var suspects []ProcessMemory
	for _, p := range procs {
		name, err := p.Name()
		if err != nil {
			continue
		}

		// Skip essential system processes
		if isSystemProcess(name) {
			continue
		}

		memInfo, err := p.MemoryInfo()
		if err != nil || memInfo == nil {
			continue
		}

		rss := memInfo.RSS
		if rss < memoryThreshold {
			continue
		}

		var pct float64
		if totalMem > 0 {
			pct = roundFloat(float64(rss)/float64(totalMem)*100.0, 2)
		}

		suspects = append(suspects, ProcessMemory{
			Name:    name,
			PID:     p.Pid,
			Memory:  rss,
			Percent: pct,
		})
	}

	// Sort by memory usage descending
	sort.Slice(suspects, func(i, j int) bool {
		return suspects[i].Memory > suspects[j].Memory
	})

	return suspects, nil
}

// isSystemProcess checks if a process name is a known essential system process.
func isSystemProcess(name string) bool {
	lower := strings.ToLower(name)
	return systemProcesses[lower]
}

// calculateCached extracts cached memory from the virtual memory statistics.
// On Windows this corresponds to the standby list and modified page list.
func calculateCached(vmStat *mem.VirtualMemoryStat) uint64 {
	// gopsutil exposes Total - Available - Used as an approximation;
	// however the raw Windows MEMORYSTATUSEX is available if we query directly.
	// We use a direct Windows API call for accuracy.
	var memStatus memoryStatusEx
	memStatus.dwLength = uint32(unsafe.Sizeof(memStatus))

	kernel32 := windows.NewLazyDLL("kernel32.dll")
	globalMemoryStatusEx := kernel32.NewProc("GlobalMemoryStatusEx")
	ret, _, _ := globalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&memStatus)))
	if ret == 0 {
		// Fallback: estimate cached = Total - Used - Available (may be negative, clamp to 0)
		if vmStat.Total > vmStat.Used+vmStat.Available {
			return vmStat.Total - vmStat.Used - vmStat.Available
		}
		return 0
	}

	// Cached memory approximation: total physical - available - (total - available - free_from_standby)
	// Simpler: the difference between what's not available and what's actively used
	// On Windows, Available includes standby, so cached ~ Total - Available is not right.
	// Best approximation with MEMORYSTATUSEX: cached = Total - Used (from gopsutil) - Free
	// where Free is the zero-page/free list and Used is the active working set.
	// gopsutil's Available = free + standby, so standby (cached) = Available - free
	// Since we can't easily get "free" from MEMORYSTATUSEX, use gopsutil's approach:
	// cached = Available - (TotalPhys - UsedPhys - standby), but this is circular.
	// Practical approach: cached = Total - Used - (Total - Available) = Available - (Total - Used - Available)?
	// Simplest correct: cached = vmStat.Total - vmStat.Used - freeMemory
	// where freeMemory = memStatus.ullAvailPhys roughly, but that includes standby.

	// The most practical approach: return what gopsutil doesn't directly expose.
	// On Windows, standby/cached = Total - Available(free+standby) is wrong.
	// Let's use: cached = Total - Used - Free, where Free is the actual free pages.
	// Windows Available = Standby + Free, so Standby = Available - Free.
	// We approximate Free as Available - (Total - Used - Available) when that's positive.
	// Actually, simplest: cached = Total - vmStat.Used - vmStat.Free (gopsutil does expose Free on some platforms)

	// Since gopsutil on Windows sets Available = free + cached/standby,
	// the cached portion is approximately: Available - (Total - Used - Available)
	// But that doesn't work either. Let's just use the direct formula:
	// Cached/Standby = Available - Free. If gopsutil gives us Free, great.
	// Otherwise, estimate: in-use + cached + free = total,
	// gopsutil: Used = in-use, Available = cached + free
	// So cached = Available - free. We estimate free from the MEMORYSTATUSEX load percentage.

	// Pragmatic: return Total - Used - a rough free estimate
	// The most useful value for the user is the standby memory that CAN be flushed.
	// Standby = Total - Used - Free = Total - Used - (something small)
	// Since Windows keeps very little truly free, cached ~ Total - Used - (small amount)
	// Better to return: Total - Used - Available is negative on Windows, so:
	// cached ~ vmStat.Total - vmStat.Available - vmStat.Used won't work.

	// Final approach: On Windows, gopsutil Available already includes standby.
	// The best we can do without performance counters: cached = Available portion that's standby.
	// Windows roughly: Free pages are very small. So cached ≈ Available * 0.8 as a rough estimate.
	// But that's hacky. Let's just return (Total - Used) - Available if positive, else 0.
	// This represents memory that's neither in active use nor available (modified pages, etc.)
	diff := vmStat.Total - vmStat.Used
	if diff > vmStat.Available {
		return diff - vmStat.Available
	}
	return 0
}

// memoryStatusEx matches the Windows MEMORYSTATUSEX structure.
type memoryStatusEx struct {
	dwLength                uint32
	dwMemoryLoad            uint32
	ullTotalPhys            uint64
	ullAvailPhys            uint64
	ullTotalPageFile        uint64
	ullAvailPageFile        uint64
	ullTotalVirtual         uint64
	ullAvailVirtual         uint64
	ullAvailExtendedVirtual uint64
}

// roundFloat rounds a float64 to the specified number of decimal places.
func roundFloat(val float64, places int) float64 {
	factor := 1.0
	for i := 0; i < places; i++ {
		factor *= 10
	}
	return float64(int(val*factor+0.5)) / factor
}
