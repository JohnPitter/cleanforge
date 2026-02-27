package main

import (
	"fmt"
	"os"
	"os/user"
	"strings"

	"cleanforge/internal/cleaner"
	"cleanforge/internal/gaming"
	"cleanforge/internal/memory"
	"cleanforge/internal/network"
	"cleanforge/internal/privacy"
	"cleanforge/internal/system"
	"cleanforge/internal/toolkit"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
)

func runCLI() {
	green := color.New(color.FgHiGreen, color.Bold)
	cyan := color.New(color.FgHiCyan)
	yellow := color.New(color.FgHiYellow)
	red := color.New(color.FgHiRed)

	green.Println("\n  ██████╗██╗     ███████╗ █████╗ ███╗   ██╗███████╗ ██████╗ ██████╗  ██████╗ ███████╗")
	green.Println("  ██╔════╝██║     ██╔════╝██╔══██╗████╗  ██║██╔════╝██╔═══██╗██╔══██╗██╔════╝ ██╔════╝")
	green.Println("  ██║     ██║     █████╗  ███████║██╔██╗ ██║█████╗  ██║   ██║██████╔╝██║  ███╗█████╗  ")
	green.Println("  ██║     ██║     ██╔══╝  ██╔══██║██║╚═██║██╔══╝  ██║   ██║██╔══██╗██║   ██║██╔══╝  ")
	green.Println("  ╚██████╗███████╗███████╗██║  ██║██║ ╚████║██║     ╚██████╔╝██║  ██║╚██████╔╝███████╗")
	green.Println("   ╚═════╝╚══════╝╚══════╝╚═╝  ╚═╝╚═╝  ╚═══╝╚═╝      ╚═════╝ ╚═╝  ╚═╝ ╚═════╝ ╚══════╝")
	fmt.Println()
	cyan.Println("  Ultimate Windows Performance Suite v1.0.0")
	fmt.Println("  ─────────────────────────────────────────────")
	fmt.Println()

	for {
		prompt := promptui.Select{
			Label: "What would you like to do?",
			Items: []string{
				"🖥️  System Info",
				"🧹 Quick Clean (Safe files only)",
				"🧹 Full Scan & Clean",
				"🎮 Game Boost",
				"🌐 Network Optimizer",
				"🛡️  Privacy Protection",
				"🔧 System Tools",
				"💾 Memory Optimizer",
				"❌ Exit",
			},
			Size: 9,
		}

		i, _, err := prompt.Run()
		if err != nil {
			break
		}

		fmt.Println()

		switch i {
		case 0:
			cliSystemInfo(cyan, yellow)
		case 1:
			cliQuickClean(green, yellow, red)
		case 2:
			cliFullClean(green, yellow, red)
		case 3:
			cliGameBoost(green, yellow, red)
		case 4:
			cliNetwork(green, yellow)
		case 5:
			cliPrivacy(green, yellow)
		case 6:
			cliToolkit(green, yellow, red)
		case 7:
			cliMemory(green, yellow)
		case 8:
			green.Println("  Thanks for using CleanForge! 🔥")
			os.Exit(0)
		}
		fmt.Println()
	}
}

func cliSystemInfo(cyan, yellow *color.Color) {
	info, err := system.GetSystemInfo()
	if err != nil {
		color.Red("  Error: %v", err)
		return
	}

	cyan.Println("  ═══ System Information ═══")
	fmt.Printf("  OS:          %s\n", info.OS)
	fmt.Printf("  Hostname:    %s\n", info.Hostname)
	fmt.Printf("  CPU:         %s (%dC/%dT)\n", info.CPUModel, info.CPUCores, info.CPUThreads)
	fmt.Printf("  CPU Usage:   %.1f%%\n", info.CPUUsage)
	fmt.Printf("  RAM:         %.1f GB / %.1f GB (%.1f%%)\n",
		float64(info.RAMUsed)/1024/1024/1024,
		float64(info.RAMTotal)/1024/1024/1024,
		info.RAMUsage)
	fmt.Printf("  GPU:         %s (Driver: %s)\n", info.GPUName, info.GPUDriver)
	fmt.Printf("  Uptime:      %s\n", info.Uptime)

	score := info.HealthScore
	scoreColor := color.New(color.FgHiGreen)
	if score < 60 {
		scoreColor = color.New(color.FgHiYellow)
	}
	if score < 40 {
		scoreColor = color.New(color.FgHiRed)
	}
	scoreColor.Printf("  Health:      %d/100\n", score)

	fmt.Println("\n  Disks:")
	for _, d := range info.Disks {
		fmt.Printf("    %s  %.1f GB free / %.1f GB total (%.0f%% used)\n",
			d.Drive,
			float64(d.Free)/1024/1024/1024,
			float64(d.Total)/1024/1024/1024,
			d.UsagePercent)
	}
}

func cliQuickClean(green, yellow, red *color.Color) {
	u, _ := user.Current()
	username := "User"
	if u != nil {
		username = u.Username
		if idx := strings.LastIndex(username, "\\"); idx >= 0 {
			username = username[idx+1:]
		}
	}

	c := cleaner.NewCleaner(username)
	yellow.Println("  Scanning safe categories...")
	result, err := c.Scan()
	if err != nil {
		red.Printf("  Scan error: %v\n", err)
		return
	}

	safeIDs := []string{}
	for _, cat := range result.Categories {
		if cat.Risk == "safe" && cat.Size > 0 {
			safeIDs = append(safeIDs, cat.ID)
			fmt.Printf("  ✓ %s: %s\n", cat.Name, formatBytesHuman(cat.Size))
		}
	}

	if len(safeIDs) == 0 {
		green.Println("  System is already clean!")
		return
	}

	prompt := promptui.Prompt{
		Label:     "Clean these safe categories",
		IsConfirm: true,
	}

	_, err = prompt.Run()
	if err != nil {
		return
	}

	cleanResult, err := c.Clean(safeIDs)
	if err != nil {
		red.Printf("  Clean error: %v\n", err)
		return
	}
	green.Printf("  ✓ Freed %s (%d files deleted)\n", formatBytesHuman(cleanResult.FreedSpace), cleanResult.DeletedFiles)
}

func cliFullClean(green, yellow, red *color.Color) {
	u, _ := user.Current()
	username := "User"
	if u != nil {
		username = u.Username
		if idx := strings.LastIndex(username, "\\"); idx >= 0 {
			username = username[idx+1:]
		}
	}

	c := cleaner.NewCleaner(username)
	yellow.Println("  Scanning all categories...")
	result, err := c.Scan()
	if err != nil {
		red.Printf("  Scan error: %v\n", err)
		return
	}

	var items []string
	var ids []string
	for _, cat := range result.Categories {
		if cat.Size > 0 {
			riskLabel := "SAFE"
			if cat.Risk == "low" {
				riskLabel = "LOW RISK"
			} else if cat.Risk == "medium" {
				riskLabel = "MEDIUM"
			}
			items = append(items, fmt.Sprintf("[%s] %s - %s", riskLabel, cat.Name, formatBytesHuman(cat.Size)))
			ids = append(ids, cat.ID)
		}
	}

	if len(items) == 0 {
		green.Println("  System is already clean!")
		return
	}

	fmt.Println("  Found:")
	for _, item := range items {
		fmt.Printf("    %s\n", item)
	}
	yellow.Printf("\n  Total: %s in %d files\n", formatBytesHuman(result.TotalSize), result.TotalFiles)

	prompt := promptui.Prompt{
		Label:     "Clean all categories",
		IsConfirm: true,
	}

	_, err = prompt.Run()
	if err != nil {
		return
	}

	cleanResult, err := c.Clean(ids)
	if err != nil {
		red.Printf("  Clean error: %v\n", err)
		return
	}
	green.Printf("  ✓ Freed %s (%d files deleted)\n", formatBytesHuman(cleanResult.FreedSpace), cleanResult.DeletedFiles)
}

func cliGameBoost(green, yellow, red *color.Color) {
	gb := gaming.NewGameBooster()

	gpuInfo, _ := gb.DetectGPU()
	if gpuInfo != nil {
		fmt.Printf("  GPU: %s (%s)\n", gpuInfo.Name, strings.ToUpper(gpuInfo.Vendor))
	}

	profiles := gb.GetProfiles()
	items := make([]string, len(profiles)+1)
	for i, p := range profiles {
		items[i] = fmt.Sprintf("%s - %s", p.Name, p.Description)
	}
	items[len(profiles)] = "Restore Original Settings"

	prompt := promptui.Select{
		Label: "Select Game Profile",
		Items: items,
		Size:  7,
	}

	i, _, err := prompt.Run()
	if err != nil {
		return
	}

	if i == len(profiles) {
		yellow.Println("  Restoring original settings...")
		if err := gb.RestoreAll(); err != nil {
			red.Printf("  Error: %v\n", err)
		} else {
			green.Println("  ✓ All settings restored!")
		}
		return
	}

	yellow.Printf("  Applying %s profile...\n", profiles[i].Name)
	if err := gb.ApplyProfile(profiles[i].ID); err != nil {
		red.Printf("  Error: %v\n", err)
	} else {
		green.Printf("  ✓ %s profile applied!\n", profiles[i].Name)
	}
}

func cliNetwork(green, yellow *color.Color) {
	prompt := promptui.Select{
		Label: "Network Optimizer",
		Items: []string{
			"Show Network Status",
			"Set DNS - Cloudflare (1.1.1.1)",
			"Set DNS - Google (8.8.8.8)",
			"Reset DNS to DHCP",
			"Disable Nagle (reduce latency)",
			"Enable Nagle (restore default)",
			"Flush Network Stack",
			"Ping Test",
			"Back",
		},
		Size: 9,
	}

	i, _, err := prompt.Run()
	if err != nil {
		return
	}

	switch i {
	case 0:
		status, err := network.GetNetworkStatus()
		if err != nil {
			color.Red("  Error: %v", err)
			return
		}
		fmt.Printf("  Adapter: %s\n  IP: %s\n  DNS: %s\n  Nagle: %v\n",
			status.Adapter, status.IPAddress, status.CurrentDNS,
			map[bool]string{true: "Disabled (fast)", false: "Enabled (default)"}[status.NagleDisabled])
	case 1:
		presets := network.GetDNSPresets()
		network.SetDNS(presets[0])
		green.Println("  ✓ DNS set to Cloudflare (1.1.1.1)")
	case 2:
		presets := network.GetDNSPresets()
		network.SetDNS(presets[1])
		green.Println("  ✓ DNS set to Google (8.8.8.8)")
	case 3:
		network.ResetDNS()
		green.Println("  ✓ DNS reset to DHCP")
	case 4:
		network.DisableNagle()
		green.Println("  ✓ Nagle disabled (lower latency)")
	case 5:
		network.EnableNagle()
		green.Println("  ✓ Nagle enabled (default)")
	case 6:
		yellow.Println("  Flushing network stack...")
		output, _ := network.FlushNetwork()
		green.Println("  ✓ Network flushed")
		fmt.Println(output)
	case 7:
		yellow.Println("  Pinging 8.8.8.8...")
		latency, err := network.PingTest("8.8.8.8")
		if err != nil {
			color.Red("  Error: %v", err)
		} else {
			green.Printf("  ✓ Latency: %.1fms\n", latency)
		}
	}
}

func cliPrivacy(green, yellow *color.Color) {
	prompt := promptui.Select{
		Label: "Privacy Protection",
		Items: []string{
			"Show Privacy Status",
			"Apply All Protections",
			"Restore All Defaults",
			"Back",
		},
		Size: 4,
	}

	i, _, err := prompt.Run()
	if err != nil {
		return
	}

	switch i {
	case 0:
		tweaks, _ := privacy.GetPrivacyTweaks()
		for _, t := range tweaks {
			status := "❌"
			if t.Applied {
				status = "✅"
			}
			fmt.Printf("  %s %s - %s\n", status, t.Name, t.Description)
		}
	case 1:
		yellow.Println("  Applying all privacy protections...")
		privacy.ApplyAll()
		green.Println("  ✓ All protections applied!")
	case 2:
		yellow.Println("  Restoring defaults...")
		privacy.RestoreAll()
		green.Println("  ✓ All restored to defaults")
	}
}

func cliToolkit(green, yellow, red *color.Color) {
	if !toolkit.IsAdmin() {
		red.Println("  ⚠ Some tools require administrator privileges.")
		red.Println("  Run CleanForge as Administrator for full access.")
		fmt.Println()
	}

	prompt := promptui.Select{
		Label: "System Tools",
		Items: []string{
			"Run SFC (System File Checker)",
			"Run DISM Repair",
			"Rebuild Icon Cache",
			"Rebuild Font Cache",
			"Reset Windows Search",
			"Repair Windows Update",
			"Remove Bloatware",
			"Back",
		},
		Size: 8,
	}

	i, _, err := prompt.Run()
	if err != nil {
		return
	}

	var result *toolkit.ToolResult
	switch i {
	case 0:
		yellow.Println("  Running SFC (this may take a while)...")
		result, err = toolkit.RunSFC()
	case 1:
		yellow.Println("  Running DISM (this may take a while)...")
		result, err = toolkit.RunDISM()
	case 2:
		yellow.Println("  Rebuilding icon cache...")
		result, err = toolkit.RebuildIconCache()
	case 3:
		yellow.Println("  Rebuilding font cache...")
		result, err = toolkit.RebuildFontCache()
	case 4:
		yellow.Println("  Resetting Windows Search...")
		result, err = toolkit.ResetWindowsSearch()
	case 5:
		yellow.Println("  Repairing Windows Update...")
		result, err = toolkit.RepairWindowsUpdate()
	case 6:
		apps, _ := toolkit.GetBloatwareApps()
		var installed []toolkit.BloatwareApp
		for _, a := range apps {
			if a.Installed {
				installed = append(installed, a)
			}
		}
		if len(installed) == 0 {
			green.Println("  No bloatware found!")
			return
		}
		fmt.Println("  Found bloatware:")
		pkgs := []string{}
		for _, a := range installed {
			fmt.Printf("    • %s (%s)\n", a.Name, a.Publisher)
			pkgs = append(pkgs, a.PackageName)
		}
		confirm := promptui.Prompt{Label: "Remove all bloatware", IsConfirm: true}
		if _, err := confirm.Run(); err == nil {
			result, err = toolkit.RemoveBloatware(pkgs)
		}
		return
	case 7:
		return
	}

	if err != nil {
		red.Printf("  Error: %v\n", err)
		return
	}
	if result != nil {
		if result.Success {
			green.Printf("  ✓ %s completed\n", result.Name)
		} else {
			red.Printf("  ✗ %s had errors\n", result.Name)
		}
	}
}

func cliMemory(green, yellow *color.Color) {
	status, err := memory.GetMemoryStatus()
	if err != nil {
		color.Red("  Error: %v", err)
		return
	}

	fmt.Printf("  RAM: %s / %s (%.1f%%)\n",
		formatBytesHuman(int64(status.Used)),
		formatBytesHuman(int64(status.Total)),
		status.UsagePercent)

	if len(status.TopProcesses) > 0 {
		fmt.Println("\n  Top Memory Consumers:")
		for i, p := range status.TopProcesses {
			if i >= 5 {
				break
			}
			fmt.Printf("    %d. %s - %s (%.1f%%)\n", i+1, p.Name, formatBytesHuman(int64(p.Memory)), p.Percent)
		}
	}

	prompt := promptui.Select{
		Label: "Memory Optimizer",
		Items: []string{"Flush Memory (Trim Working Sets)", "Detect Memory Leaks", "Back"},
		Size:  3,
	}

	i, _, err := prompt.Run()
	if err != nil {
		return
	}

	switch i {
	case 0:
		yellow.Println("  Flushing memory...")
		memory.FlushStandbyList()
		green.Println("  ✓ Memory flushed!")
	case 1:
		leaks, _ := memory.DetectMemoryLeaks()
		if len(leaks) == 0 {
			green.Println("  No suspicious memory usage found!")
		} else {
			fmt.Println("  Suspicious processes (>500MB):")
			for _, p := range leaks {
				fmt.Printf("    • %s (PID %d) - %s\n", p.Name, p.PID, formatBytesHuman(int64(p.Memory)))
			}
		}
	}
}

func formatBytesHuman(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	b := float64(bytes)
	i := 0
	for b >= 1024 && i < len(units)-1 {
		b /= 1024
		i++
	}
	return fmt.Sprintf("%.1f %s", b, units[i])
}
