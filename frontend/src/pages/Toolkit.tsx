import { useState, useEffect } from "react";
import { motion, AnimatePresence } from "framer-motion";
import {
  Wrench,
  ShieldCheck,
  ShieldAlert,
  HardDrive,
  Paintbrush,
  Type,
  Search,
  Package,
  Trash2,
  Loader2,
  CheckCircle2,
  XCircle,
  AlertTriangle,
  Lock,
} from "lucide-react";

interface ToolResult {
  name: string;
  success: boolean;
  output: string;
  errors: string[];
}

interface BloatwareApp {
  name: string;
  packageName: string;
  publisher: string;
  installed: boolean;
}

const tools = [
  { id: "sfc", name: "System File Checker", desc: "Scan and repair system files (sfc /scannow)", icon: ShieldCheck, color: "text-forge-accent", admin: true },
  { id: "dism", name: "DISM Repair", desc: "Repair Windows image (DISM /RestoreHealth)", icon: HardDrive, color: "text-forge-info", admin: true },
  { id: "icon_cache", name: "Rebuild Icon Cache", desc: "Fix broken or blank desktop icons", icon: Paintbrush, color: "text-forge-purple", admin: false },
  { id: "font_cache", name: "Rebuild Font Cache", desc: "Fix missing or corrupted fonts", icon: Type, color: "text-forge-warning", admin: true },
  { id: "search_reset", name: "Reset Windows Search", desc: "Fix 100% disk usage from indexing", icon: Search, color: "text-red-400", admin: true },
  { id: "wu_repair", name: "Repair Windows Update", desc: "Fix stuck Windows updates", icon: Package, color: "text-green-400", admin: true },
];

export default function Toolkit() {
  const [running, setRunning] = useState<string | null>(null);
  const [results, setResults] = useState<Map<string, ToolResult>>(new Map());
  const [bloatware, setBloatware] = useState<BloatwareApp[]>([]);
  const [loadingBloat, setLoadingBloat] = useState(false);
  const [selectedBloat, setSelectedBloat] = useState<Set<string>>(new Set());
  const [removingBloat, setRemovingBloat] = useState(false);
  const [showBloatware, setShowBloatware] = useState(false);
  const [isAdmin, setIsAdmin] = useState<boolean | null>(null);
  const [expandedTool, setExpandedTool] = useState<string | null>(null);

  useEffect(() => {
    // @ts-ignore - Wails bindings
    window.go?.main?.App?.GetIsAdmin?.().then((v: boolean) => setIsAdmin(v));
  }, []);

  async function runTool(id: string) {
    setRunning(id);
    try {
      let result: ToolResult;
      switch (id) {
        // @ts-ignore
        case "sfc": result = await window.go.main.App.RunSFC(); break;
        // @ts-ignore
        case "dism": result = await window.go.main.App.RunDISM(); break;
        // @ts-ignore
        case "icon_cache": result = await window.go.main.App.RebuildIconCache(); break;
        // @ts-ignore
        case "font_cache": result = await window.go.main.App.RebuildFontCache(); break;
        // @ts-ignore
        case "search_reset": result = await window.go.main.App.ResetWindowsSearch(); break;
        // @ts-ignore
        case "wu_repair": result = await window.go.main.App.RepairWindowsUpdate(); break;
        default: return;
      }
      setResults(new Map(results.set(id, result)));
      setExpandedTool(id);
    } catch (e) {
      console.error("Tool failed:", e);
    }
    setRunning(null);
  }

  async function loadBloatware() {
    setLoadingBloat(true);
    setShowBloatware(true);
    try {
      // @ts-ignore
      const apps = await window.go.main.App.GetBloatwareApps();
      setBloatware(apps?.filter((a: BloatwareApp) => a.installed) || []);
    } catch {}
    setLoadingBloat(false);
  }

  async function removeBloatware() {
    if (selectedBloat.size === 0) return;
    setRemovingBloat(true);
    try {
      // @ts-ignore
      await window.go.main.App.RemoveBloatware(Array.from(selectedBloat));
      await loadBloatware();
      setSelectedBloat(new Set());
    } catch {}
    setRemovingBloat(false);
  }

  function toggleBloat(pkg: string) {
    const next = new Set(selectedBloat);
    if (next.has(pkg)) next.delete(pkg);
    else next.add(pkg);
    setSelectedBloat(next);
  }

  return (
    <div className="p-6 space-y-6 overflow-y-auto h-full">
      <div>
        <h2 className="text-2xl font-bold text-forge-text flex items-center gap-2">
          <Wrench className="w-6 h-6 text-forge-accent" /> Toolkit
        </h2>
        <p className="text-sm text-forge-muted">System repair tools and bloatware removal</p>
      </div>

      {/* Admin Warning Banner */}
      {isAdmin === false && (
        <div className="flex items-center gap-3 p-3 bg-forge-warning/10 border border-forge-warning/30 rounded-xl">
          <ShieldAlert className="w-5 h-5 text-forge-warning shrink-0" />
          <div>
            <p className="text-sm font-semibold text-forge-warning">Not running as Administrator</p>
            <p className="text-xs text-forge-muted">
              Most repair tools require admin privileges. Right-click CleanForge and select "Run as administrator".
            </p>
          </div>
        </div>
      )}

      {/* System Tools Grid */}
      <div>
        <h3 className="text-sm font-semibold text-forge-muted uppercase tracking-wider mb-3">
          Repair Tools
        </h3>
        <div className="grid grid-cols-3 gap-3">
          {tools.map((tool) => {
            const result = results.get(tool.id);
            const needsAdmin = tool.admin && isAdmin === false;
            return (
              <motion.button
                key={tool.id}
                whileHover={{ scale: 1.02 }}
                whileTap={{ scale: 0.98 }}
                onClick={() => runTool(tool.id)}
                disabled={running !== null || needsAdmin}
                className={`text-left p-4 bg-forge-card border rounded-xl transition-all disabled:opacity-50 ${
                  needsAdmin
                    ? "border-forge-border/50 opacity-60"
                    : "border-forge-border hover:border-forge-accent/30"
                }`}
              >
                <div className="flex items-center gap-2 mb-2">
                  {running === tool.id ? (
                    <Loader2 className={`w-5 h-5 ${tool.color} animate-spin`} />
                  ) : (
                    <tool.icon className={`w-5 h-5 ${tool.color}`} />
                  )}
                  <span className="text-sm font-bold text-forge-text">{tool.name}</span>
                  {tool.admin && (
                    <span title="Requires Administrator" className="ml-auto">
                      <Lock className="w-3 h-3 text-forge-muted" />
                    </span>
                  )}
                </div>
                <p className="text-xs text-forge-muted">{tool.desc}</p>
                {needsAdmin && !result && (
                  <div className="mt-2 flex items-center gap-1 text-[10px] text-forge-warning">
                    <AlertTriangle className="w-3 h-3" />
                    Requires admin
                  </div>
                )}
                {result && (
                  <div className={`mt-2 flex items-center gap-1 text-[10px] ${result.success ? "text-forge-accent" : "text-forge-danger"}`}>
                    {result.success ? <CheckCircle2 className="w-3 h-3" /> : <XCircle className="w-3 h-3" />}
                    {result.success ? "Completed" : result.output || "Errors found"}
                  </div>
                )}
              </motion.button>
            );
          })}
        </div>
      </div>

      {/* Tool Output Detail */}
      <AnimatePresence>
        {expandedTool && results.get(expandedTool) && (
          <motion.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: "auto" }}
            exit={{ opacity: 0, height: 0 }}
            className="bg-forge-card border border-forge-border rounded-xl p-4 space-y-2"
          >
            <div className="flex items-center justify-between">
              <h4 className="text-sm font-semibold text-forge-text">
                {results.get(expandedTool)!.name} — Output
              </h4>
              <button
                onClick={() => setExpandedTool(null)}
                className="text-xs text-forge-muted hover:text-forge-text transition-colors"
              >
                Close
              </button>
            </div>
            {results.get(expandedTool)!.output && (
              <pre className="text-xs text-forge-muted bg-forge-bg rounded-lg p-3 overflow-x-auto whitespace-pre-wrap max-h-40 overflow-y-auto">
                {results.get(expandedTool)!.output}
              </pre>
            )}
            {results.get(expandedTool)!.errors?.length > 0 && (
              <div className="space-y-1">
                {results.get(expandedTool)!.errors.map((err, i) => (
                  <div key={i} className="flex items-start gap-2 text-xs text-forge-danger">
                    <XCircle className="w-3 h-3 mt-0.5 shrink-0" />
                    <span>{err}</span>
                  </div>
                ))}
              </div>
            )}
          </motion.div>
        )}
      </AnimatePresence>

      {/* Bloatware Remover */}
      <div>
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-semibold text-forge-muted uppercase tracking-wider flex items-center gap-2">
            <Package className="w-4 h-4" /> Bloatware Remover
          </h3>
          {!showBloatware && (
            <button
              onClick={loadBloatware}
              className="text-xs text-forge-accent hover:text-forge-accent-dim transition-colors"
            >
              Scan for bloatware
            </button>
          )}
        </div>

        {loadingBloat && (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="w-5 h-5 text-forge-accent animate-spin" />
          </div>
        )}

        {showBloatware && !loadingBloat && (
          <>
            {bloatware.length === 0 ? (
              <div className="text-center py-6 text-forge-muted text-sm">
                <CheckCircle2 className="w-8 h-8 mx-auto mb-2 opacity-30" />
                No bloatware found. Your system is clean!
              </div>
            ) : (
              <>
                <div className="space-y-1.5 max-h-48 overflow-y-auto">
                  {bloatware.map((app) => (
                    <div
                      key={app.packageName}
                      onClick={() => toggleBloat(app.packageName)}
                      className={`flex items-center gap-3 p-2.5 rounded-lg cursor-pointer transition-colors border ${
                        selectedBloat.has(app.packageName)
                          ? "bg-forge-danger/10 border-forge-danger/30"
                          : "bg-forge-surface border-forge-border"
                      }`}
                    >
                      <input
                        type="checkbox"
                        checked={selectedBloat.has(app.packageName)}
                        readOnly
                        className="accent-forge-danger"
                      />
                      <div className="flex-1">
                        <p className="text-xs font-medium text-forge-text">{app.name}</p>
                        <p className="text-[10px] text-forge-muted">{app.publisher}</p>
                      </div>
                    </div>
                  ))}
                </div>
                {selectedBloat.size > 0 && (
                  <motion.button
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    whileHover={{ scale: 1.02 }}
                    whileTap={{ scale: 0.98 }}
                    onClick={removeBloatware}
                    disabled={removingBloat}
                    className="mt-3 flex items-center gap-2 px-4 py-2 bg-forge-danger/10 border border-forge-danger/30 text-forge-danger rounded-lg text-xs font-semibold hover:bg-forge-danger/20 transition-colors"
                  >
                    {removingBloat ? <Loader2 className="w-3 h-3 animate-spin" /> : <Trash2 className="w-3 h-3" />}
                    Remove {selectedBloat.size} app{selectedBloat.size > 1 ? "s" : ""}
                  </motion.button>
                )}
              </>
            )}
          </>
        )}
      </div>
    </div>
  );
}
