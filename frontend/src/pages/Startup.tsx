import { useState, useEffect } from "react";
import { motion } from "framer-motion";
import { Rocket, ToggleLeft, ToggleRight, AlertTriangle, Loader2, ArrowUpDown } from "lucide-react";

interface StartupItem {
  name: string;
  path: string;
  publisher: string;
  impact: string;
  enabled: boolean;
  location: string;
}

const impactColors: Record<string, { bg: string; text: string }> = {
  high: { bg: "bg-forge-danger/10", text: "text-forge-danger" },
  medium: { bg: "bg-forge-warning/10", text: "text-forge-warning" },
  low: { bg: "bg-forge-accent/10", text: "text-forge-accent" },
  unknown: { bg: "bg-forge-card", text: "text-forge-muted" },
};

export default function Startup() {
  const [items, setItems] = useState<StartupItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [toggling, setToggling] = useState<string | null>(null);
  const [sortBy, setSortBy] = useState<"impact" | "name">("impact");

  useEffect(() => {
    loadItems();
  }, []);

  async function loadItems() {
    setLoading(true);
    try {
      // @ts-ignore
      const data = await window.go.main.App.GetStartupItems();
      setItems(data || []);
    } catch (e) {
      console.error("Failed to load startup items:", e);
    }
    setLoading(false);
  }

  async function toggleItem(item: StartupItem) {
    setToggling(item.name);
    try {
      if (item.enabled) {
        // @ts-ignore
        await window.go.main.App.DisableStartupItem(item);
      } else {
        // @ts-ignore
        await window.go.main.App.EnableStartupItem(item);
      }
      await loadItems();
    } catch (e) {
      console.error("Toggle failed:", e);
    }
    setToggling(null);
  }

  const impactOrder: Record<string, number> = { high: 0, medium: 1, low: 2, unknown: 3 };
  const sorted = [...items].sort((a, b) => {
    if (sortBy === "impact") return (impactOrder[a.impact] ?? 3) - (impactOrder[b.impact] ?? 3);
    return a.name.localeCompare(b.name);
  });

  const enabledCount = items.filter((i) => i.enabled).length;
  const highImpact = items.filter((i) => i.impact === "high" && i.enabled).length;

  return (
    <div className="p-6 space-y-6 overflow-y-auto h-full">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-forge-text flex items-center gap-2">
            <Rocket className="w-6 h-6 text-forge-accent" /> Startup Manager
          </h2>
          <p className="text-sm text-forge-muted">
            Control which programs run when Windows starts
          </p>
        </div>
        <button
          onClick={() => setSortBy(sortBy === "impact" ? "name" : "impact")}
          className="flex items-center gap-1.5 px-3 py-1.5 text-xs text-forge-muted bg-forge-card border border-forge-border rounded-lg hover:text-forge-text transition-colors"
        >
          <ArrowUpDown className="w-3 h-3" />
          Sort by {sortBy === "impact" ? "Name" : "Impact"}
        </button>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-3 gap-3">
        <div className="bg-forge-card border border-forge-border rounded-xl p-3 text-center">
          <p className="text-2xl font-bold text-forge-text">{items.length}</p>
          <p className="text-xs text-forge-muted">Total Programs</p>
        </div>
        <div className="bg-forge-card border border-forge-border rounded-xl p-3 text-center">
          <p className="text-2xl font-bold text-forge-accent">{enabledCount}</p>
          <p className="text-xs text-forge-muted">Enabled</p>
        </div>
        <div className="bg-forge-card border border-forge-border rounded-xl p-3 text-center">
          <p className={`text-2xl font-bold ${highImpact > 0 ? "text-forge-danger" : "text-forge-accent"}`}>
            {highImpact}
          </p>
          <p className="text-xs text-forge-muted">High Impact</p>
        </div>
      </div>

      {highImpact > 2 && (
        <div className="flex items-center gap-2 p-3 bg-forge-warning/10 border border-forge-warning/30 rounded-lg">
          <AlertTriangle className="w-4 h-4 text-forge-warning shrink-0" />
          <p className="text-xs text-forge-warning">
            You have {highImpact} high-impact programs at startup. Disabling them can significantly speed up boot time.
          </p>
        </div>
      )}

      {/* Items List */}
      {loading ? (
        <div className="flex items-center justify-center py-16">
          <Loader2 className="w-6 h-6 text-forge-accent animate-spin" />
        </div>
      ) : (
        <div className="space-y-1.5">
          {sorted.map((item, i) => {
            const impact = impactColors[item.impact] || impactColors.unknown;
            return (
              <motion.div
                key={`${item.name}-${item.location}`}
                initial={{ opacity: 0, x: -10 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ delay: i * 0.02 }}
                className="flex items-center gap-3 p-3 bg-forge-card border border-forge-border rounded-lg hover:border-forge-border transition-colors"
              >
                <button
                  onClick={() => toggleItem(item)}
                  disabled={toggling === item.name}
                  className="shrink-0"
                >
                  {toggling === item.name ? (
                    <Loader2 className="w-6 h-6 text-forge-muted animate-spin" />
                  ) : item.enabled ? (
                    <ToggleRight className="w-6 h-6 text-forge-accent" />
                  ) : (
                    <ToggleLeft className="w-6 h-6 text-forge-muted" />
                  )}
                </button>

                <div className="flex-1 min-w-0">
                  <p className={`text-sm font-medium ${item.enabled ? "text-forge-text" : "text-forge-muted"}`}>
                    {item.name}
                  </p>
                  <p className="text-[10px] text-forge-muted truncate">{item.path}</p>
                </div>

                <span className={`text-[10px] px-2 py-0.5 rounded ${impact.bg} ${impact.text}`}>
                  {item.impact}
                </span>

                <span className="text-[10px] text-forge-muted">{item.location.replace("_", " ")}</span>
              </motion.div>
            );
          })}
        </div>
      )}
    </div>
  );
}
