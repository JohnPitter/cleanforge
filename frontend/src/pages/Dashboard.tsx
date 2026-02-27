import { useState, useEffect } from "react";
import { motion } from "framer-motion";
import { Cpu, MemoryStick, HardDrive, Monitor, Clock, Zap } from "lucide-react";
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

function SkeletonPulse({ className = "" }: { className?: string }) {
  return <div className={`animate-pulse bg-forge-border/40 rounded ${className}`} />;
}

function DashboardSkeleton() {
  return (
    <div className="p-6 space-y-6 overflow-y-auto h-full">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-forge-text">Dashboard</h2>
          <p className="text-sm text-forge-muted">System health overview</p>
        </div>
        <div className="flex items-center gap-2 text-xs text-forge-muted">
          <div className="w-2 h-2 rounded-full bg-forge-accent animate-pulse" />
          Loading...
        </div>
      </div>

      <div className="grid grid-cols-12 gap-5">
        {/* Health Score Skeleton */}
        <div className="col-span-3 bg-forge-card border border-forge-border rounded-xl p-5 flex flex-col items-center justify-center">
          <div className="relative w-36 h-36 flex items-center justify-center">
            <div className="w-28 h-28 rounded-full border-[6px] border-forge-border/40 animate-pulse" />
          </div>
          <SkeletonPulse className="h-3 w-20 mt-3" />
        </div>

        {/* Stat Cards Skeleton */}
        <div className="col-span-9 grid grid-cols-3 gap-4">
          {Array.from({ length: 6 }).map((_, i) => (
            <div key={i} className="bg-forge-card border border-forge-border rounded-xl p-4">
              <div className="flex items-center gap-2 mb-3">
                <SkeletonPulse className="w-4 h-4 rounded" />
                <SkeletonPulse className="h-3 w-12" />
              </div>
              <SkeletonPulse className="h-7 w-24 mb-2" />
              <SkeletonPulse className="h-3 w-32" />
              {i < 2 && <SkeletonPulse className="h-1.5 w-full mt-3 rounded-full" />}
            </div>
          ))}
        </div>
      </div>

      {/* Storage Partitions Skeleton */}
      <div>
        <div className="flex items-center gap-2 mb-3">
          <SkeletonPulse className="w-4 h-4 rounded" />
          <SkeletonPulse className="h-3 w-32" />
        </div>
        <div className="grid grid-cols-4 gap-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="bg-forge-card border border-forge-border rounded-xl p-4">
              <div className="flex items-center justify-between mb-2">
                <SkeletonPulse className="h-4 w-10" />
                <SkeletonPulse className="h-3 w-8" />
              </div>
              <SkeletonPulse className="h-2 w-full rounded-full mb-2" />
              <div className="flex justify-between">
                <SkeletonPulse className="h-2.5 w-16" />
                <SkeletonPulse className="h-2.5 w-16" />
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
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

  if (!info) return <DashboardSkeleton />;

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
          {/* CPU Card */}
          <StatCard
            icon={Cpu}
            label="CPU"
            value={`${cpuUsage.toFixed(1)}%`}
            subValue={info?.cpuModel?.split("@")[0]?.trim() ?? "Loading..."}
            percentage={cpuUsage}
          >
            <div className="flex gap-3 text-[10px] text-forge-muted">
              <span>{info?.cpuCores ?? 0} Cores</span>
              <span>{info?.cpuThreads ?? 0} Threads</span>
            </div>
          </StatCard>

          {/* RAM Card */}
          <StatCard
            icon={MemoryStick}
            label="RAM"
            value={info ? formatBytes(info.ramUsed) : "..."}
            subValue={info ? `of ${formatBytes(info.ramTotal)}` : ""}
            percentage={ramUsage}
          >
            {info?.ramModules?.length ? (
              <div className="space-y-1">
                {info.ramModules.map((mod, i) => (
                  <div key={i} className="text-[10px] text-forge-muted">
                    <span className="text-forge-text font-medium">{mod.manufacturer}</span>
                    {" "}{formatBytes(mod.capacity)} {mod.speed > 0 && `${mod.speed}MHz`} {mod.formFactor}
                  </div>
                ))}
              </div>
            ) : null}
          </StatCard>

          {/* GPU Card */}
          <StatCard
            icon={Monitor}
            label="GPU"
            value={info?.gpuName ?? "..."}
            subValue={info?.gpuDriver ? `Driver: ${info.gpuDriver}` : ""}
            color="text-forge-purple"
          >
            {info?.gpus?.length ? (
              <div className="space-y-1">
                {info.gpus.map((gpu, i) => (
                  <div key={i} className="text-[10px] text-forge-muted">
                    {gpu.vram > 0 && <span>VRAM: <span className="text-forge-text font-medium">{formatBytes(gpu.vram)}</span></span>}
                  </div>
                ))}
              </div>
            ) : null}
          </StatCard>

          {/* Uptime Card */}
          <StatCard
            icon={Clock}
            label="Uptime"
            value={info?.uptime ?? "..."}
            color="text-forge-info"
          />

          {/* Storage Card */}
          <StatCard
            icon={HardDrive}
            label="Storage"
            value={info?.physDisks?.length ? `${info.physDisks.length} Drive${info.physDisks.length > 1 ? "s" : ""}` : "..."}
            color="text-forge-warning"
          >
            {info?.physDisks?.length ? (
              <div className="space-y-1">
                {info.physDisks.map((d, i) => (
                  <div key={i} className="text-[10px] text-forge-muted">
                    <span className="text-forge-text font-medium">{d.model}</span>
                    {" "}{formatBytes(d.size)} {d.mediaType}
                  </div>
                ))}
              </div>
            ) : null}
          </StatCard>

          {/* System Card */}
          <StatCard
            icon={Zap}
            label="System"
            value={info?.hostname ?? "..."}
            subValue={info?.os ?? ""}
            color="text-forge-accent"
          />
        </div>
      </div>

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
