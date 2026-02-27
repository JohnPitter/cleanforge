import { useState, useEffect } from "react";
import { motion } from "framer-motion";
import { Cpu, MemoryStick, HardDrive, Activity, Monitor, Clock, Zap } from "lucide-react";
import StatCard from "../components/StatCard";
import HealthScore from "../components/HealthScore";

interface RAMModule {
  manufacturer: string;
  capacity: number;
  speed: number;
  partNumber: string;
  formFactor: string;
  slot: string;
}

interface GPUDetail {
  name: string;
  driver: string;
  vram: number;
}

interface PhysDisk {
  model: string;
  size: number;
  mediaType: string;
  interface: string;
}

interface SystemInfo {
  os: string;
  hostname: string;
  cpuModel: string;
  cpuCores: number;
  cpuThreads: number;
  cpuUsage: number;
  ramTotal: number;
  ramUsed: number;
  ramUsage: number;
  ramModules: RAMModule[];
  gpuName: string;
  gpuDriver: string;
  gpus: GPUDetail[];
  disks: { drive: string; total: number; used: number; free: number; usagePercent: number; fsType: string }[];
  physDisks: PhysDisk[];
  uptime: string;
  healthScore: number;
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + " " + sizes[i];
}

export default function Dashboard() {
  const [info, setInfo] = useState<SystemInfo | null>(null);
  const [cpuUsage, setCpuUsage] = useState(0);
  const [ramUsage, setRamUsage] = useState(0);

  useEffect(() => {
    loadSystemInfo();
    const interval = setInterval(refreshMetrics, 3000);
    return () => clearInterval(interval);
  }, []);

  async function loadSystemInfo() {
    try {
      // @ts-ignore - Wails bindings
      const data = await window.go.main.App.GetSystemInfo();
      setInfo(data);
      setCpuUsage(data.cpuUsage);
      setRamUsage(data.ramUsage);
    } catch (e) {
      console.error("Failed to load system info:", e);
    }
  }

  async function refreshMetrics() {
    try {
      // @ts-ignore
      const data = await window.go.main.App.GetSystemInfo();
      if (data) {
        setCpuUsage(data.cpuUsage);
        setRamUsage(data.ramUsage);
        setInfo(data);
      }
    } catch {}
  }

  return (
    <div className="p-6 space-y-6 overflow-y-auto h-full">
      <motion.div
        initial={{ opacity: 0, y: -10 }}
        animate={{ opacity: 1, y: 0 }}
        className="flex items-center justify-between"
      >
        <div>
          <h2 className="text-2xl font-bold text-forge-text">Dashboard</h2>
          <p className="text-sm text-forge-muted">
            System health overview
          </p>
        </div>
        <div className="flex items-center gap-2 text-xs text-forge-muted">
          <div className="w-2 h-2 rounded-full bg-forge-accent animate-pulse" />
          Live monitoring
        </div>
      </motion.div>

      <div className="grid grid-cols-12 gap-5">
        {/* Health Score */}
        <motion.div
          initial={{ opacity: 0, scale: 0.95 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ delay: 0.1 }}
          className="col-span-3 bg-forge-card border border-forge-border rounded-xl p-5 flex flex-col items-center justify-center"
        >
          <HealthScore score={info?.healthScore ?? 0} />
          <p className="text-xs text-forge-muted mt-3">System Health</p>
        </motion.div>

        {/* Stats Grid */}
        <div className="col-span-9 grid grid-cols-3 gap-4">
          <StatCard
            icon={Cpu}
            label="CPU"
            value={`${cpuUsage.toFixed(1)}%`}
            subValue={info?.cpuModel?.split("@")[0]?.trim() ?? "Loading..."}
            percentage={cpuUsage}
          />
          <StatCard
            icon={MemoryStick}
            label="RAM"
            value={info ? formatBytes(info.ramUsed) : "..."}
            subValue={info ? `of ${formatBytes(info.ramTotal)}` : ""}
            percentage={ramUsage}
          />
          <StatCard
            icon={Monitor}
            label="GPU"
            value={info?.gpuName?.split(" ").slice(0, 3).join(" ") ?? "..."}
            subValue={info?.gpuDriver ? `Driver: ${info.gpuDriver}` : ""}
            color="text-forge-purple"
          />
          <StatCard
            icon={Clock}
            label="Uptime"
            value={info?.uptime ?? "..."}
            color="text-forge-info"
          />
          <StatCard
            icon={Activity}
            label="Cores / Threads"
            value={info ? `${info.cpuCores}C / ${info.cpuThreads}T` : "..."}
            color="text-forge-warning"
          />
          <StatCard
            icon={Zap}
            label="System"
            value={info?.hostname ?? "..."}
            subValue={info?.os ?? ""}
            color="text-forge-accent"
          />
        </div>
      </div>

      {/* Hardware Details */}
      <motion.div
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.2 }}
        className="grid grid-cols-2 gap-5"
      >
        {/* CPU Details */}
        <div className="bg-forge-card border border-forge-border rounded-xl p-5">
          <h3 className="text-sm font-semibold text-forge-muted uppercase tracking-wider mb-3 flex items-center gap-2">
            <Cpu className="w-4 h-4" /> Processor
          </h3>
          <p className="text-sm text-forge-text font-medium">{info?.cpuModel ?? "..."}</p>
          <div className="mt-2 flex gap-4 text-xs text-forge-muted">
            <span>{info?.cpuCores ?? 0} Cores</span>
            <span>{info?.cpuThreads ?? 0} Threads</span>
          </div>
        </div>

        {/* GPU Details */}
        <div className="bg-forge-card border border-forge-border rounded-xl p-5">
          <h3 className="text-sm font-semibold text-forge-muted uppercase tracking-wider mb-3 flex items-center gap-2">
            <Monitor className="w-4 h-4" /> Graphics
          </h3>
          {info?.gpus?.length ? (
            <div className="space-y-3">
              {info.gpus.map((gpu, i) => (
                <div key={i}>
                  <p className="text-sm text-forge-text font-medium">{gpu.name}</p>
                  <div className="mt-1 flex gap-4 text-xs text-forge-muted">
                    {gpu.vram > 0 && <span>VRAM: {formatBytes(gpu.vram)}</span>}
                    {gpu.driver && <span>Driver: {gpu.driver}</span>}
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-forge-muted">No GPU detected</p>
          )}
        </div>

        {/* RAM Modules */}
        <div className="bg-forge-card border border-forge-border rounded-xl p-5">
          <h3 className="text-sm font-semibold text-forge-muted uppercase tracking-wider mb-3 flex items-center gap-2">
            <MemoryStick className="w-4 h-4" /> Memory Modules
          </h3>
          {info?.ramModules?.length ? (
            <div className="space-y-2">
              {info.ramModules.map((mod, i) => (
                <div key={i} className="flex items-center justify-between text-sm">
                  <div>
                    <span className="text-forge-text font-medium">{mod.slot}</span>
                    <span className="text-forge-muted ml-2 text-xs">
                      {mod.manufacturer}{mod.partNumber ? ` — ${mod.partNumber.trim()}` : ""}
                    </span>
                  </div>
                  <div className="flex gap-3 text-xs text-forge-muted">
                    <span>{formatBytes(mod.capacity)}</span>
                    {mod.speed > 0 && <span>{mod.speed} MHz</span>}
                    <span>{mod.formFactor}</span>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-forge-muted">No module data available</p>
          )}
        </div>

        {/* Physical Disks */}
        <div className="bg-forge-card border border-forge-border rounded-xl p-5">
          <h3 className="text-sm font-semibold text-forge-muted uppercase tracking-wider mb-3 flex items-center gap-2">
            <HardDrive className="w-4 h-4" /> Physical Drives
          </h3>
          {info?.physDisks?.length ? (
            <div className="space-y-2">
              {info.physDisks.map((d, i) => (
                <div key={i} className="flex items-center justify-between text-sm">
                  <span className="text-forge-text font-medium">{d.model}</span>
                  <div className="flex gap-3 text-xs text-forge-muted">
                    <span>{formatBytes(d.size)}</span>
                    <span>{d.mediaType}</span>
                    <span>{d.interface}</span>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-forge-muted">No drive data available</p>
          )}
        </div>
      </motion.div>

      {/* Disk Usage */}
      <motion.div
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.3 }}
      >
        <h3 className="text-sm font-semibold text-forge-muted uppercase tracking-wider mb-3 flex items-center gap-2">
          <HardDrive className="w-4 h-4" /> Storage Partitions
        </h3>
        <div className="grid grid-cols-4 gap-4">
          {info?.disks?.map((disk) => (
            <div
              key={disk.drive}
              className="bg-forge-card border border-forge-border rounded-xl p-4"
            >
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm font-bold text-forge-text">
                  {disk.drive}
                </span>
                <span
                  className={`text-xs font-mono ${
                    disk.usagePercent > 90
                      ? "text-forge-danger"
                      : disk.usagePercent > 70
                        ? "text-forge-warning"
                        : "text-forge-accent"
                  }`}
                >
                  {disk.usagePercent.toFixed(0)}%
                </span>
              </div>
              <div className="h-2 bg-forge-bg rounded-full overflow-hidden mb-2">
                <motion.div
                  initial={{ width: 0 }}
                  animate={{ width: `${disk.usagePercent}%` }}
                  transition={{ duration: 1 }}
                  className={`h-full rounded-full ${
                    disk.usagePercent > 90
                      ? "bg-forge-danger"
                      : disk.usagePercent > 70
                        ? "bg-forge-warning"
                        : "bg-forge-accent"
                  }`}
                />
              </div>
              <div className="flex justify-between text-[10px] text-forge-muted">
                <span>{formatBytes(disk.used)} used</span>
                <span>{formatBytes(disk.free)} free</span>
              </div>
            </div>
          ))}
        </div>
      </motion.div>
    </div>
  );
}
