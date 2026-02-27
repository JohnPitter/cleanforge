<div align="center">

# CleanForge

### The Ultimate Windows Performance Suite

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://golang.org)
[![Wails](https://img.shields.io/badge/Wails-v2-DF0000?style=for-the-badge&logo=webassembly&logoColor=white)](https://wails.io)
[![React](https://img.shields.io/badge/React-18-61DAFB?style=for-the-badge&logo=react&logoColor=black)](https://reactjs.org)
[![License](https://img.shields.io/badge/License-MIT-00FF88?style=for-the-badge)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Windows-0078D6?style=for-the-badge&logo=windows&logoColor=white)](https://www.microsoft.com/windows)

**System Cleanup** &bull; **Game Boost** &bull; **Startup Manager** &bull; **Network Optimizer** &bull; **Privacy Guard** &bull; **System Toolkit**

[Installation](#-installation) &bull; [Features](#-features) &bull; [Game Profiles](#-game-profiles) &bull; [CLI Mode](#-cli-mode) &bull; [Contributing](#-contributing)

---

</div>

## Overview

**CleanForge** is an open-source, all-in-one Windows performance optimization tool built with Go and React. It combines system cleanup, gaming optimization, startup management, network tuning, privacy protection, and system repair tools into a single modern application.

Whether your PC is a "bomb" about to stop from junk and bad configs, or you want to squeeze every last FPS for competitive gaming — CleanForge has you covered.

> No bloatware. No ads. No telemetry. Just pure performance.

---

## Requirements

| Requirement | Version |
|---|---|
| **Operating System** | Windows 10/11 (64-bit) |
| **Go** | 1.24+ |
| **Node.js** | 18+ |
| **Wails CLI** | v2.x |

---

## Installation

### Download Release

Download the latest `.exe` from the [Releases](https://github.com/JohnPitter/cleanforge/releases) page and run it. No installation required.

### Build from Source

```bash
# Clone the repository
git clone https://github.com/JohnPitter/cleanforge.git
cd cleanforge

# Install frontend dependencies
cd frontend && npm install && cd ..

# Build the application
wails build

# The executable will be at build/bin/cleanforge.exe
```

### Development Mode

```bash
wails dev
```

---

## Features

### System Cleanup

| Category | What it cleans | Risk Level |
|---|---|---|
| Windows Temp | `C:\Windows\Temp` | Safe |
| User Temp | `AppData\Local\Temp` | Safe |
| Recycle Bin | All drives | Safe |
| Browser Cache | Chrome, Edge, Firefox | Safe |
| npm Cache | `AppData\Roaming\npm-cache` | Low |
| Maven Cache | `.m2\repository` | Low |
| Gradle Cache | `.gradle\caches` | Low |
| Go Cache | `go clean -cache` | Low |
| Windows Update | `SoftwareDistribution\Download` | Low |
| Windows Logs | `C:\Windows\Logs` | Low |
| Prefetch | `C:\Windows\Prefetch` | Low |
| Thumbnails | Explorer thumbnail cache | Safe |

**Smart scanning** calculates sizes before cleaning and shows a detailed breakdown.

---

### Game Boost

Comprehensive gaming optimization with **one-click profiles** and **full restore capability**.

#### Peripheral Optimization

- **Mouse:** Disable pointer acceleration, raw input mode, no smooth scroll
- **Keyboard:** Minimum repeat delay, maximum repeat rate, disable Sticky/Filter/Toggle Keys

#### GPU Maximum Performance

| GPU Vendor | Optimizations |
|---|---|
| **NVIDIA** | Power Management: Max Performance, Low Latency: Ultra, Threaded Optimization |
| **AMD** | Anti-Lag, ULPS disabled, Power Profile: Performance |
| **Intel** | Performance mode, adaptive vsync off |

#### System Tweaks

- Ultimate Performance power plan (hidden Windows plan)
- Disable core parking (use all CPU cores)
- Disable HPET (reduce latency)
- Timer Resolution 0.5ms
- Disable SysMain/SuperFetch
- Disable Game DVR, Game Bar, Game Mode
- Disable fullscreen optimizations
- Kill bloatware processes

> All changes are backed up and can be restored with one click.

---

### Game Profiles

| Profile | Best For | Key Features |
|---|---|---|
| **Competitive FPS** | Valorant, CS2, Apex | Raw mouse, zero latency, kill all bloat |
| **Open World** | Cyberpunk, GTA V, Elden Ring | Stutter-free, GPU max, no indexing |
| **MOBA/Strategy** | LoL, Dota 2, AoE | Network optimized, fast keyboard |
| **Racing/Sim** | Forza, F1, iRacing | GPU max, stable frametime |
| **Casual** | Minecraft, Stardew Valley | Light optimization, balanced |
| **Nuclear Mode** | Any game | ALL tweaks enabled, maximum aggression |

---

### Startup Manager

- List all startup programs with **impact rating** (High/Medium/Low)
- One-click **enable/disable** for each program
- Reads from Registry (HKCU/HKLM), Startup Folder, and Task Scheduler
- Identifies heavy impact programs automatically

---

### Network Optimizer

- **DNS Presets:** Cloudflare (1.1.1.1), Google (8.8.8.8), OpenDNS, Quad9
- **Nagle Algorithm:** Disable for reduced network latency
- **Network Flush:** DNS flush + Winsock reset + TCP/IP stack reset
- **Ping Test:** Built-in latency measurement

---

### Privacy Guard

Disable Windows telemetry, tracking, advertising, and data collection:

- Disable telemetry data collection
- Disable activity history & location tracking
- Disable advertising ID
- Disable Cortana & Bing Search in Start Menu
- Disable feedback & tailored experiences
- Block telemetry domains via hosts file
- Disable WiFi Sense & error reporting

All protections can be applied individually or all at once, with full restore capability.

---

### System Toolkit

- **System File Checker (SFC)** — scan and repair corrupted system files
- **DISM Repair** — repair Windows component store
- **Bloatware Remover** — remove pre-installed Windows apps (Candy Crush, TikTok, etc.)
- **Rebuild Icon Cache** — fix broken desktop icons
- **Rebuild Font Cache** — fix corrupted fonts
- **Reset Windows Search** — fix 100% disk usage from indexing
- **Repair Windows Update** — fix stuck updates

---

### Memory Optimizer

- Real-time RAM monitoring with top process breakdown
- Flush standby memory (trim working sets)
- Detect memory leaks (processes >500MB)

---

### System Monitor

- Real-time CPU, RAM, GPU temperature monitoring
- Disk usage across all drives
- Health Score (0-100) based on system metrics
- Built-in benchmark (CPU/RAM/Disk scoring)
- Thermal throttling detection and alerts

---

## CLI Mode

CleanForge also includes a full-featured interactive CLI:

```bash
# Launch CLI mode
cleanforge.exe --cli

# Available commands:
# - System Info
# - Quick Clean (safe files only)
# - Full Scan & Clean
# - Game Boost (with profile selection)
# - Network Optimizer
# - Privacy Protection
# - System Tools
# - Memory Optimizer
```

---

## Tech Stack

| Component | Technology |
|---|---|
| **Backend** | Go 1.24+ |
| **Frontend** | React 18 + TypeScript |
| **UI Framework** | Wails v2 (native WebView) |
| **Styling** | Tailwind CSS v4 |
| **Animations** | Framer Motion |
| **Icons** | Lucide React |
| **Charts** | Recharts |

---

## Project Structure

```
cleanforge/
├── main.go                  # Entry point (GUI or CLI mode)
├── app.go                   # Wails bridge (Go <-> React)
├── cli.go                   # Interactive CLI mode
├── internal/
│   ├── system/              # System info (CPU, RAM, Disk, GPU)
│   ├── cleaner/             # File cleanup engine
│   ├── gaming/              # Game Boost (profiles, tweaks, GPU)
│   │   └── profiles/        # Predefined game profiles
│   ├── startup/             # Startup manager
│   ├── network/             # Network optimizer (DNS, Nagle)
│   ├── toolkit/             # System repair tools
│   ├── privacy/             # Privacy & telemetry controls
│   ├── memory/              # Memory optimizer
│   ├── monitor/             # System monitoring & benchmark
│   └── backup/              # State backup & restore
├── frontend/
│   └── src/
│       ├── components/      # Reusable UI components
│       ├── pages/           # Dashboard, Cleaner, GameBoost, etc.
│       └── hooks/           # Custom React hooks
└── build/                   # Build configuration
```

---

## Contributing

Contributions are welcome!

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

<div align="center">

**Built with Go + React + Wails**

Made by [JohnPitter](https://github.com/JohnPitter)

</div>
