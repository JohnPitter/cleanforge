import { useState, useEffect } from "react";
import { motion } from "framer-motion";
import {
  Wifi,
  Globe,
  Zap,
  RotateCcw,
  CheckCircle2,
  Loader2,
  Activity,
} from "lucide-react";

interface NetworkStatus {
  currentDns: string;
  nagleDisabled: boolean;
  adapter: string;
  ipAddress: string;
  gateway: string;
}

interface DNSPreset {
  id: string;
  name: string;
  primary: string;
  secondary: string;
  description: string;
}

const dnsPresets: DNSPreset[] = [
  { id: "cloudflare", name: "Cloudflare", primary: "1.1.1.1", secondary: "1.0.0.1", description: "Fastest, privacy-focused" },
  { id: "google", name: "Google", primary: "8.8.8.8", secondary: "8.8.4.4", description: "Reliable, low latency" },
  { id: "opendns", name: "OpenDNS", primary: "208.67.222.222", secondary: "208.67.220.220", description: "Family-safe option" },
  { id: "quad9", name: "Quad9", primary: "9.9.9.9", secondary: "149.112.112.112", description: "Security-focused" },
];

function getActivePresetName(currentDns: string): string | null {
  if (!currentDns) return null;
  for (const preset of dnsPresets) {
    if (currentDns.includes(preset.primary)) {
      return preset.name;
    }
  }
  return null;
}

export default function Network() {
  const [status, setStatus] = useState<NetworkStatus | null>(null);
  const [applying, setApplying] = useState<string | null>(null);
  const [flushing, setFlushing] = useState(false);
  const [flushResult, setFlushResult] = useState<string | null>(null);
  const [pingResult, setPingResult] = useState<number | null>(null);
  const [pinging, setPinging] = useState(false);

  useEffect(() => {
    loadStatus();
  }, []);

  async function loadStatus() {
    try {
      // @ts-ignore
      const data = await window.go.main.App.GetNetworkStatus();
      setStatus(data);
    } catch {
      // Even if the backend call fails, show the status card with empty/fallback values
      setStatus({ currentDns: "", nagleDisabled: false, adapter: "", ipAddress: "", gateway: "" });
    }
  }

  async function setDNS(preset: DNSPreset) {
    setApplying(preset.id);
    try {
      // @ts-ignore
      await window.go.main.App.SetDNS(preset);
      await loadStatus();
    } catch (e) {
      console.error("DNS change failed:", e);
    }
    setApplying(null);
  }

  async function resetDNS() {
    setApplying("reset");
    try {
      // @ts-ignore
      await window.go.main.App.ResetDNS();
      await loadStatus();
    } catch {}
    setApplying(null);
  }

  async function toggleNagle() {
    setApplying("nagle");
    try {
      if (status?.nagleDisabled) {
        // @ts-ignore
        await window.go.main.App.EnableNagle();
      } else {
        // @ts-ignore
        await window.go.main.App.DisableNagle();
      }
      await loadStatus();
    } catch {}
    setApplying(null);
  }

  async function flushNetwork() {
    setFlushing(true);
    setFlushResult(null);
    try {
      // @ts-ignore
      const result = await window.go.main.App.FlushNetwork();
      setFlushResult(result);
    } catch {}
    setFlushing(false);
  }

  async function runPing() {
    setPinging(true);
    setPingResult(null);
    try {
      // @ts-ignore
      const ms = await window.go.main.App.PingTest("8.8.8.8");
      setPingResult(ms);
    } catch {}
    setPinging(false);
  }

  return (
    <div className="p-6 space-y-6 overflow-y-auto h-full">
      <div>
        <h2 className="text-2xl font-bold text-forge-text flex items-center gap-2">
          <Wifi className="w-6 h-6 text-forge-accent" /> Network Optimizer
        </h2>
        <p className="text-sm text-forge-muted">Optimize DNS, reduce latency, and fix network issues</p>
      </div>

      {/* Status Card */}
      {status && (
        <motion.div
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          className="bg-forge-card border border-forge-border rounded-xl p-4 grid grid-cols-4 gap-4"
        >
          <div>
            <p className="text-[10px] text-forge-muted uppercase tracking-wider">Adapter</p>
            <p className="text-sm font-medium text-forge-text truncate">{status.adapter || "N/A"}</p>
          </div>
          <div>
            <p className="text-[10px] text-forge-muted uppercase tracking-wider">IP Address</p>
            <p className="text-sm font-mono text-forge-text">{status.ipAddress || "N/A"}</p>
          </div>
          <div>
            <p className="text-[10px] text-forge-muted uppercase tracking-wider">Current DNS</p>
            {(() => {
              const presetName = getActivePresetName(status.currentDns);
              if (presetName) {
                return (
                  <div>
                    <p className="text-sm font-semibold text-forge-accent">{presetName}</p>
                    <p className="text-[10px] font-mono text-forge-muted">{status.currentDns}</p>
                  </div>
                );
              }
              return (
                <p className="text-sm font-mono text-forge-text">
                  {status.currentDns || "DHCP (Automatic)"}
                </p>
              );
            })()}
          </div>
          <div>
            <p className="text-[10px] text-forge-muted uppercase tracking-wider">Nagle</p>
            <p className={`text-sm font-semibold ${status.nagleDisabled ? "text-forge-accent" : "text-forge-warning"}`}>
              {status.nagleDisabled ? "Disabled (Fast)" : "Enabled (Default)"}
            </p>
          </div>
        </motion.div>
      )}

      {/* DNS Presets */}
      <div>
        <h3 className="text-sm font-semibold text-forge-muted uppercase tracking-wider mb-3 flex items-center gap-2">
          <Globe className="w-4 h-4" /> DNS Presets
        </h3>
        <div className="grid grid-cols-2 gap-3">
          {dnsPresets.map((preset) => {
            const isActive = status?.currentDns
              ? status.currentDns.includes(preset.primary)
              : false;
            return (
              <motion.button
                key={preset.id}
                whileHover={{ scale: 1.01 }}
                whileTap={{ scale: 0.99 }}
                onClick={() => setDNS(preset)}
                disabled={applying !== null}
                className={`text-left p-4 bg-forge-card border rounded-xl transition-all disabled:opacity-50 ${
                  isActive
                    ? "border-forge-accent shadow-[0_0_12px_rgba(16,185,129,0.15)]"
                    : "border-forge-border hover:border-forge-accent/30"
                }`}
              >
                <div className="flex items-center justify-between mb-1">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-bold text-forge-text">{preset.name}</span>
                    {isActive && (
                      <span className="flex items-center gap-1 px-1.5 py-0.5 text-[9px] font-semibold uppercase tracking-wider bg-forge-accent/15 text-forge-accent rounded-full">
                        <CheckCircle2 className="w-2.5 h-2.5" /> Active
                      </span>
                    )}
                  </div>
                  {applying === preset.id && <Loader2 className="w-3 h-3 animate-spin text-forge-accent" />}
                </div>
                <p className="text-xs text-forge-muted mb-2">{preset.description}</p>
                <p className="text-[10px] font-mono text-forge-muted">
                  {preset.primary} | {preset.secondary}
                </p>
              </motion.button>
            );
          })}
        </div>
        <button
          onClick={resetDNS}
          disabled={applying !== null}
          className="mt-2 flex items-center gap-1.5 px-3 py-1.5 text-xs text-forge-muted hover:text-forge-text transition-colors"
        >
          <RotateCcw className="w-3 h-3" /> Reset to DHCP
        </button>
      </div>

      {/* Quick Actions */}
      <div className="grid grid-cols-3 gap-3">
        <motion.button
          whileHover={{ scale: 1.02 }}
          whileTap={{ scale: 0.98 }}
          onClick={toggleNagle}
          disabled={applying !== null}
          className="p-4 bg-forge-card border border-forge-border rounded-xl hover:border-forge-accent/30 transition-all text-center"
        >
          <Zap className="w-5 h-5 text-forge-warning mx-auto mb-2" />
          <p className="text-xs font-semibold text-forge-text">
            {status?.nagleDisabled ? "Enable Nagle" : "Disable Nagle"}
          </p>
          <p className="text-[10px] text-forge-muted mt-1">Reduces network latency</p>
        </motion.button>

        <motion.button
          whileHover={{ scale: 1.02 }}
          whileTap={{ scale: 0.98 }}
          onClick={flushNetwork}
          disabled={flushing}
          className="p-4 bg-forge-card border border-forge-border rounded-xl hover:border-forge-accent/30 transition-all text-center"
        >
          {flushing ? (
            <Loader2 className="w-5 h-5 text-forge-info mx-auto mb-2 animate-spin" />
          ) : (
            <RotateCcw className="w-5 h-5 text-forge-info mx-auto mb-2" />
          )}
          <p className="text-xs font-semibold text-forge-text">Flush Network</p>
          <p className="text-[10px] text-forge-muted mt-1">DNS + Winsock + TCP reset</p>
        </motion.button>

        <motion.button
          whileHover={{ scale: 1.02 }}
          whileTap={{ scale: 0.98 }}
          onClick={runPing}
          disabled={pinging}
          className="p-4 bg-forge-card border border-forge-border rounded-xl hover:border-forge-accent/30 transition-all text-center"
        >
          {pinging ? (
            <Loader2 className="w-5 h-5 text-forge-accent mx-auto mb-2 animate-spin" />
          ) : (
            <Activity className="w-5 h-5 text-forge-accent mx-auto mb-2" />
          )}
          <p className="text-xs font-semibold text-forge-text">Ping Test</p>
          {pingResult !== null ? (
            <p className="text-[10px] text-forge-accent mt-1">{pingResult.toFixed(1)}ms</p>
          ) : (
            <p className="text-[10px] text-forge-muted mt-1">Test latency to 8.8.8.8</p>
          )}
        </motion.button>
      </div>

      {flushResult && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          className="bg-forge-accent/10 border border-forge-accent/30 rounded-lg p-3 flex items-center gap-2"
        >
          <CheckCircle2 className="w-4 h-4 text-forge-accent shrink-0" />
          <p className="text-xs text-forge-accent">Network flush completed successfully</p>
        </motion.div>
      )}
    </div>
  );
}
