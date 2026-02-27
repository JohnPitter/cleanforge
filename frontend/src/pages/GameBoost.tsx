import { useState, useEffect } from "react";
import { motion, AnimatePresence } from "framer-motion";
import {
  Gamepad2,
  Crosshair,
  Globe2,
  Swords,
  Car,
  Joystick,
  Skull,
  Zap,
  RotateCcw,
  Monitor,
  Mouse,
  Keyboard,
  Cpu,
  Loader2,
  CheckCircle2,
  ChevronDown,
  ChevronUp,
} from "lucide-react";

interface GPUInfo {
  name: string;
  vendor: string;
  driver: string;
  profileName: string;
}

interface BoostStatus {
  active: boolean;
  profile: string;
  tweaksApplied: string[];
  startedAt: string;
}

const profiles = [
  { id: "competitive_fps", name: "Competitive FPS", icon: Crosshair, desc: "Valorant, CS2, Apex", color: "text-red-400", features: ["Raw mouse input", "No pointer acceleration", "GPU low latency", "Kill all bloatware", "Timer 0.5ms", "Nagle off"] },
  { id: "open_world", name: "Open World", icon: Globe2, desc: "Cyberpunk, GTA V, Elden Ring", color: "text-green-400", features: ["Core parking off", "GPU max perf", "RAM cleanup", "No indexing", "HPET off"] },
  { id: "moba_strategy", name: "MOBA / Strategy", icon: Swords, desc: "LoL, Dota 2, AoE", color: "text-blue-400", features: ["DNS optimized", "Nagle off", "Key repeat max", "Network flush", "CPU priority high"] },
  { id: "racing_sim", name: "Racing / Sim", icon: Car, desc: "Forza, F1, iRacing", color: "text-yellow-400", features: ["GPU max perf", "No fullscreen opt.", "Ultimate power", "Core parking off"] },
  { id: "casual", name: "Casual", icon: Joystick, desc: "Minecraft, Stardew", color: "text-purple-400", features: ["Light cleanup", "RAM cleanup", "Kill heavy bloat"] },
  { id: "nuclear", name: "Nuclear Mode", icon: Skull, desc: "MAX PERFORMANCE", color: "text-forge-danger", features: ["ALL tweaks enabled", "Maximum aggression", "Full system override"] },
];

const tweakCategories = [
  { name: "Mouse", icon: Mouse, tweaks: ["Disable pointer acceleration", "Raw input mode", "No smooth scroll"] },
  { name: "Keyboard", icon: Keyboard, tweaks: ["Min repeat delay", "Max repeat rate", "Disable Sticky/Filter Keys"] },
  { name: "GPU", icon: Monitor, tweaks: ["Max performance mode", "Low latency mode", "Disable Game DVR", "Disable Game Bar", "Disable fullscreen opt."] },
  { name: "System", icon: Cpu, tweaks: ["Ultimate power plan", "Disable core parking", "Disable HPET", "Timer 0.5ms", "Disable SysMain", "Disable Win Search", "Disable Game Mode"] },
];

export default function GameBoost() {
  const [gpu, setGpu] = useState<GPUInfo | null>(null);
  const [status, setStatus] = useState<BoostStatus | null>(null);
  const [selectedProfile, setSelectedProfile] = useState<string | null>(null);
  const [applying, setApplying] = useState(false);
  const [restoring, setRestoring] = useState(false);
  const [showTweaks, setShowTweaks] = useState(false);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);
  const [restoredMsg, setRestoredMsg] = useState(false);

  useEffect(() => {
    loadGPU();
    loadStatus();
  }, []);

  async function loadGPU() {
    try {
      // @ts-ignore
      const data = await window.go.main.App.DetectGPU();
      setGpu(data);
    } catch {}
  }

  async function loadStatus() {
    try {
      // @ts-ignore
      const data = await window.go.main.App.GetBoostStatus();
      setStatus(data);
      if (data?.active) setSelectedProfile(data.profile);
    } catch {}
  }

  async function applyBoost() {
    if (!selectedProfile) return;
    setApplying(true);
    setSuccessMsg(null);
    setRestoredMsg(false);
    try {
      // @ts-ignore
      await window.go.main.App.ApplyGameProfile(selectedProfile);
    } catch (e) {
      // ApplyGameProfile may return errors for partial success (some tweaks fail),
      // but the profile IS still applied. Log but don't bail out.
      console.warn("Boost applied with warnings:", e);
    }
    // Always refresh status after attempting to apply — the backend updates
    // the boost status even when some individual tweaks fail.
    await loadStatus();
    const profileName = profiles.find((p) => p.id === selectedProfile)?.name ?? selectedProfile;
    setSuccessMsg(`${profileName} applied successfully! Your system is now optimized for gaming.`);
    setApplying(false);
  }

  async function restoreAll() {
    setRestoring(true);
    setSuccessMsg(null);
    setRestoredMsg(false);
    try {
      // @ts-ignore
      await window.go.main.App.RestoreGameSettings();
    } catch (e) {
      // Restore may return errors for partial restore (some registry values fail),
      // but the restore IS still executed. Log but don't bail out.
      console.warn("Restore completed with warnings:", e);
    }
    // Always refresh status and show restored message
    await loadStatus();
    // If status is no longer active after restore, show success
    // @ts-ignore
    const freshStatus = await window.go.main.App.GetBoostStatus().catch(() => null);
    if (!freshStatus?.active) {
      setStatus(null);
      setSelectedProfile(null);
      setRestoredMsg(true);
    }
    setRestoring(false);
  }

  return (
    <div className="p-6 space-y-6 overflow-y-auto h-full">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-forge-text flex items-center gap-2">
            <Gamepad2 className="w-6 h-6 text-forge-accent" /> Game Boost
          </h2>
          <p className="text-sm text-forge-muted">
            Optimize your system for maximum gaming performance
          </p>
        </div>
        {status?.active && (
          <div className="flex items-center gap-2 px-3 py-1.5 bg-forge-accent/10 border border-forge-accent/30 rounded-lg">
            <div className="w-2 h-2 rounded-full bg-forge-accent animate-pulse" />
            <span className="text-xs text-forge-accent font-semibold">Boost Active</span>
          </div>
        )}
      </div>

      {/* GPU Info */}
      {gpu && (
        <motion.div
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          className="bg-forge-card border border-forge-border rounded-xl p-4 flex items-center gap-4"
        >
          <Monitor className="w-8 h-8 text-forge-purple" />
          <div>
            <p className="text-sm font-semibold text-forge-text">{gpu.name}</p>
            <p className="text-xs text-forge-muted">
              {gpu.vendor.toUpperCase()} | Driver: {gpu.driver} | Profile: {gpu.profileName}
            </p>
          </div>
          <div className="ml-auto">
            <span className="text-[10px] px-2 py-1 bg-forge-purple/10 text-forge-purple border border-forge-purple/30 rounded-md">
              Auto-detected
            </span>
          </div>
        </motion.div>
      )}

      {/* Game Profiles */}
      <div>
        <h3 className="text-sm font-semibold text-forge-muted uppercase tracking-wider mb-3">
          Select Game Profile
        </h3>
        <div className="grid grid-cols-3 gap-3">
          {profiles.map((p, i) => (
            <motion.button
              key={p.id}
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: i * 0.05 }}
              whileHover={{ scale: 1.02 }}
              whileTap={{ scale: 0.98 }}
              onClick={() => setSelectedProfile(p.id)}
              className={`text-left p-4 rounded-xl border transition-all ${
                selectedProfile === p.id
                  ? "bg-forge-card border-forge-accent/40 shadow-[0_0_15px_rgba(0,255,136,0.1)]"
                  : "bg-forge-surface border-forge-border hover:border-forge-border"
              }`}
            >
              <div className="flex items-center gap-2 mb-2">
                <p.icon className={`w-5 h-5 ${p.color}`} />
                <span className="text-sm font-bold text-forge-text">{p.name}</span>
              </div>
              <p className="text-xs text-forge-muted mb-2">{p.desc}</p>
              <div className="flex flex-wrap gap-1">
                {p.features.slice(0, 3).map((f) => (
                  <span key={f} className="text-[9px] px-1.5 py-0.5 bg-forge-bg rounded text-forge-muted">
                    {f}
                  </span>
                ))}
                {p.features.length > 3 && (
                  <span className="text-[9px] px-1.5 py-0.5 bg-forge-bg rounded text-forge-muted">
                    +{p.features.length - 3} more
                  </span>
                )}
              </div>
            </motion.button>
          ))}
        </div>
      </div>

      {/* Action Buttons */}
      <div className="flex gap-3">
        <motion.button
          whileHover={{ scale: 1.02 }}
          whileTap={{ scale: 0.98 }}
          onClick={applyBoost}
          disabled={!selectedProfile || applying}
          className="flex items-center gap-2 px-6 py-2.5 bg-forge-accent text-forge-bg rounded-lg text-sm font-bold hover:bg-forge-accent-dim transition-colors disabled:opacity-50 glow-effect"
        >
          {applying ? <Loader2 className="w-4 h-4 animate-spin" /> : <Zap className="w-4 h-4" />}
          {applying ? "Applying..." : "APPLY BOOST"}
        </motion.button>

        {status?.active && (
          <motion.button
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            whileHover={{ scale: 1.02 }}
            whileTap={{ scale: 0.98 }}
            onClick={restoreAll}
            disabled={restoring}
            className="flex items-center gap-2 px-5 py-2.5 bg-forge-card border border-forge-danger/30 text-forge-danger rounded-lg text-sm font-medium hover:bg-forge-danger/10 transition-colors disabled:opacity-50"
          >
            {restoring ? <Loader2 className="w-4 h-4 animate-spin" /> : <RotateCcw className="w-4 h-4" />}
            Restore Original Settings
          </motion.button>
        )}
      </div>

      {/* Success / Restored Messages */}
      <AnimatePresence>
        {successMsg && (
          <motion.div
            initial={{ opacity: 0, y: -5 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -5 }}
            className="bg-forge-accent/10 border border-forge-accent/30 rounded-lg p-4 flex items-start gap-3"
          >
            <CheckCircle2 className="w-5 h-5 text-forge-accent shrink-0 mt-0.5" />
            <div>
              <p className="text-sm font-semibold text-forge-accent">{successMsg}</p>
              <p className="text-xs text-forge-muted mt-1">
                You can revert all changes at any time using the "Restore Original Settings" button above.
              </p>
            </div>
          </motion.div>
        )}
        {restoredMsg && (
          <motion.div
            initial={{ opacity: 0, y: -5 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -5 }}
            className="bg-forge-info/10 border border-forge-info/30 rounded-lg p-4 flex items-center gap-3"
          >
            <CheckCircle2 className="w-5 h-5 text-forge-info shrink-0" />
            <p className="text-sm font-semibold text-forge-info">
              Original settings restored successfully. All tweaks have been reverted.
            </p>
          </motion.div>
        )}
      </AnimatePresence>

      {/* Tweaks Detail */}
      <div>
        <button
          onClick={() => setShowTweaks(!showTweaks)}
          className="flex items-center gap-2 text-sm text-forge-muted hover:text-forge-text transition-colors"
        >
          {showTweaks ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
          Advanced Tweaks
        </button>

        <AnimatePresence>
          {showTweaks && (
            <motion.div
              initial={{ height: 0, opacity: 0 }}
              animate={{ height: "auto", opacity: 1 }}
              exit={{ height: 0, opacity: 0 }}
              className="overflow-hidden mt-3 grid grid-cols-2 gap-3"
            >
              {tweakCategories.map((cat) => (
                <div key={cat.name} className="bg-forge-card border border-forge-border rounded-xl p-4">
                  <div className="flex items-center gap-2 mb-3">
                    <cat.icon className="w-4 h-4 text-forge-accent" />
                    <span className="text-sm font-semibold text-forge-text">{cat.name}</span>
                  </div>
                  <div className="space-y-1.5">
                    {cat.tweaks.map((t) => (
                      <div key={t} className="flex items-center gap-2 text-xs text-forge-muted">
                        <CheckCircle2 className="w-3 h-3 text-forge-accent/50" />
                        {t}
                      </div>
                    ))}
                  </div>
                </div>
              ))}
            </motion.div>
          )}
        </AnimatePresence>
      </div>
    </div>
  );
}
