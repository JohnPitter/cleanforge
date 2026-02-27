import { useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import {
  Trash2,
  Search,
  Sparkles,
  CheckCircle2,
  Circle,
  AlertTriangle,
  Shield,
  Loader2,
} from "lucide-react";

interface CleanCategory {
  id: string;
  name: string;
  description: string;
  icon: string;
  risk: string;
  size: number;
  fileCount: number;
}

interface ScanResult {
  categories: CleanCategory[];
  totalSize: number;
  totalFiles: number;
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + " " + sizes[i];
}

const riskColors: Record<string, { bg: string; text: string; border: string; label: string }> = {
  safe: { bg: "bg-forge-accent/10", text: "text-forge-accent", border: "border-forge-accent/30", label: "Safe" },
  low: { bg: "bg-forge-warning/10", text: "text-forge-warning", border: "border-forge-warning/30", label: "Low Risk" },
  medium: { bg: "bg-forge-danger/10", text: "text-forge-danger", border: "border-forge-danger/30", label: "Medium" },
};

export default function Cleaner() {
  const [scanResult, setScanResult] = useState<ScanResult | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [scanning, setScanning] = useState(false);
  const [cleaning, setCleaning] = useState(false);
  const [cleaned, setCleaned] = useState(false);
  const [freedSpace, setFreedSpace] = useState(0);

  async function handleScan() {
    setScanning(true);
    setCleaned(false);
    try {
      // @ts-ignore
      const result = await window.go.main.App.ScanSystem();
      setScanResult(result);
      const safeIds = new Set<string>(
        result.categories.filter((c: CleanCategory) => c.risk === "safe" && c.size > 0).map((c: CleanCategory) => c.id)
      );
      setSelected(safeIds);
    } catch (e) {
      console.error("Scan failed:", e);
    }
    setScanning(false);
  }

  async function handleClean() {
    if (selected.size === 0) return;
    setCleaning(true);
    try {
      // @ts-ignore
      const result = await window.go.main.App.CleanSystem(Array.from(selected));
      setFreedSpace(result.freedSpace);
      setCleaned(true);
      setScanResult(null);
    } catch (e) {
      console.error("Clean failed:", e);
    }
    setCleaning(false);
  }

  function toggleCategory(id: string) {
    const next = new Set(selected);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    setSelected(next);
  }

  function selectByRisk(risk: string) {
    if (!scanResult) return;
    const ids = scanResult.categories.filter((c) => c.risk === risk && c.size > 0).map((c) => c.id);
    const next = new Set(selected);
    ids.forEach((id) => next.add(id));
    setSelected(next);
  }

  const selectedSize = scanResult
    ? scanResult.categories.filter((c) => selected.has(c.id)).reduce((sum, c) => sum + c.size, 0)
    : 0;

  return (
    <div className="p-6 space-y-6 overflow-y-auto h-full">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-forge-text flex items-center gap-2">
            <Trash2 className="w-6 h-6 text-forge-accent" /> Cleaner
          </h2>
          <p className="text-sm text-forge-muted">
            Scan and clean junk files to free up disk space
          </p>
        </div>
      </div>

      {/* Action Buttons */}
      <div className="flex gap-3">
        <motion.button
          whileHover={{ scale: 1.02 }}
          whileTap={{ scale: 0.98 }}
          onClick={handleScan}
          disabled={scanning || cleaning}
          className="flex items-center gap-2 px-5 py-2.5 bg-forge-accent/10 border border-forge-accent/30 text-forge-accent rounded-lg text-sm font-semibold hover:bg-forge-accent/20 transition-colors disabled:opacity-50"
        >
          {scanning ? <Loader2 className="w-4 h-4 animate-spin" /> : <Search className="w-4 h-4" />}
          {scanning ? "Scanning..." : "Scan System"}
        </motion.button>

        {scanResult && selected.size > 0 && (
          <motion.button
            initial={{ opacity: 0, x: -10 }}
            animate={{ opacity: 1, x: 0 }}
            whileHover={{ scale: 1.02 }}
            whileTap={{ scale: 0.98 }}
            onClick={handleClean}
            disabled={cleaning}
            className="flex items-center gap-2 px-5 py-2.5 bg-forge-danger/10 border border-forge-danger/30 text-forge-danger rounded-lg text-sm font-semibold hover:bg-forge-danger/20 transition-colors disabled:opacity-50"
          >
            {cleaning ? <Loader2 className="w-4 h-4 animate-spin" /> : <Sparkles className="w-4 h-4" />}
            {cleaning ? "Cleaning..." : `Clean ${formatBytes(selectedSize)}`}
          </motion.button>
        )}
      </div>

      {/* Quick Filters */}
      {scanResult && (
        <div className="flex gap-2">
          {["safe", "low", "medium"].map((risk) => {
            const r = riskColors[risk];
            return (
              <button
                key={risk}
                onClick={() => selectByRisk(risk)}
                className={`flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium border ${r.bg} ${r.text} ${r.border} hover:opacity-80 transition-opacity`}
              >
                <Shield className="w-3 h-3" /> Select {r.label}
              </button>
            );
          })}
        </div>
      )}

      {/* Cleaned Success */}
      <AnimatePresence>
        {cleaned && (
          <motion.div
            initial={{ opacity: 0, y: -10 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0 }}
            className="bg-forge-accent/10 border border-forge-accent/30 rounded-xl p-4 flex items-center gap-3"
          >
            <CheckCircle2 className="w-6 h-6 text-forge-accent" />
            <div>
              <p className="text-sm font-semibold text-forge-accent">Cleanup Complete!</p>
              <p className="text-xs text-forge-muted">Freed {formatBytes(freedSpace)} of disk space</p>
            </div>
          </motion.div>
        )}
      </AnimatePresence>

      {/* Scan Results */}
      {scanResult && (
        <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="space-y-2">
          <div className="flex items-center justify-between mb-3">
            <p className="text-sm text-forge-muted">
              Found <span className="text-forge-text font-semibold">{formatBytes(scanResult.totalSize)}</span> in{" "}
              <span className="text-forge-text font-semibold">{scanResult.totalFiles.toLocaleString()}</span> files
            </p>
            <p className="text-xs text-forge-muted">
              Selected: {formatBytes(selectedSize)}
            </p>
          </div>

          <div className="space-y-1.5">
            {scanResult.categories
              .filter((c) => c.size > 0)
              .sort((a, b) => b.size - a.size)
              .map((cat, i) => {
                const risk = riskColors[cat.risk] || riskColors.safe;
                const isSelected = selected.has(cat.id);
                return (
                  <motion.div
                    key={cat.id}
                    initial={{ opacity: 0, x: -10 }}
                    animate={{ opacity: 1, x: 0 }}
                    transition={{ delay: i * 0.03 }}
                    onClick={() => toggleCategory(cat.id)}
                    className={`flex items-center gap-3 p-3 rounded-lg cursor-pointer transition-all border ${
                      isSelected
                        ? "bg-forge-card border-forge-accent/20"
                        : "bg-forge-surface border-forge-border hover:border-forge-border"
                    }`}
                  >
                    {isSelected ? (
                      <CheckCircle2 className="w-4 h-4 text-forge-accent shrink-0" />
                    ) : (
                      <Circle className="w-4 h-4 text-forge-muted shrink-0" />
                    )}
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <span className="text-sm font-medium text-forge-text">{cat.name}</span>
                        <span className={`text-[10px] px-1.5 py-0.5 rounded ${risk.bg} ${risk.text}`}>
                          {risk.label}
                        </span>
                      </div>
                      <p className="text-xs text-forge-muted truncate">{cat.description}</p>
                    </div>
                    <div className="text-right shrink-0">
                      <p className="text-sm font-mono font-semibold text-forge-text">{formatBytes(cat.size)}</p>
                      <p className="text-[10px] text-forge-muted">{cat.fileCount.toLocaleString()} files</p>
                    </div>
                  </motion.div>
                );
              })}
          </div>
        </motion.div>
      )}

      {/* Empty State */}
      {!scanResult && !scanning && !cleaned && (
        <div className="flex flex-col items-center justify-center py-16 text-forge-muted">
          <Search className="w-12 h-12 mb-4 opacity-30" />
          <p className="text-sm">Click "Scan System" to find junk files</p>
        </div>
      )}
    </div>
  );
}
