import { useState, useEffect } from "react";
import { motion } from "framer-motion";
import {
  ShieldCheck,
  Eye,
  EyeOff,
  Radio,
  MapPin,
  Megaphone,
  Bot,
  Search,
  MessageSquare,
  Lightbulb,
  Wifi,
  AlertTriangle,
  Loader2,
  Shield,
  CheckCircle2,
  Lock,
} from "lucide-react";

interface PrivacyTweak {
  id: string;
  name: string;
  description: string;
  category: string;
  enabled: boolean;
  applied: boolean;
}

const categoryIcons: Record<string, any> = {
  telemetry: Radio,
  tracking: Eye,
  ads: Megaphone,
  cortana: Bot,
};

const categoryLabels: Record<string, string> = {
  telemetry: "Telemetry",
  tracking: "Tracking",
  ads: "Advertising",
  cortana: "Cortana & Search",
};

export default function Privacy() {
  const [tweaks, setTweaks] = useState<PrivacyTweak[]>([]);
  const [loading, setLoading] = useState(true);
  const [applying, setApplying] = useState<string | null>(null);
  const [applyingAll, setApplyingAll] = useState(false);

  useEffect(() => {
    loadTweaks();
  }, []);

  async function loadTweaks() {
    setLoading(true);
    try {
      // @ts-ignore
      const data = await window.go.main.App.GetPrivacyTweaks();
      setTweaks(data || []);
    } catch {}
    setLoading(false);
  }

  async function toggleTweak(tweak: PrivacyTweak) {
    setApplying(tweak.id);
    try {
      // @ts-ignore
      await window.go.main.App.TogglePrivacyTweak(tweak.id);
      await loadTweaks();
    } catch {}
    setApplying(null);
  }

  async function applyAll() {
    setApplyingAll(true);
    try {
      // @ts-ignore
      await window.go.main.App.ApplyAllPrivacy();
      await loadTweaks();
    } catch {}
    setApplyingAll(false);
  }

  async function restoreAll() {
    setApplyingAll(true);
    try {
      // @ts-ignore
      await window.go.main.App.RestoreAllPrivacy();
      await loadTweaks();
    } catch {}
    setApplyingAll(false);
  }

  const appliedCount = tweaks.filter((t) => t.applied).length;
  const categories = [...new Set(tweaks.map((t) => t.category))];

  return (
    <div className="p-6 space-y-6 overflow-y-auto h-full">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-forge-text flex items-center gap-2">
            <ShieldCheck className="w-6 h-6 text-forge-accent" /> Privacy Guard
          </h2>
          <p className="text-sm text-forge-muted">
            Disable Windows telemetry, tracking, and ads
          </p>
        </div>
      </div>

      {/* Score Bar */}
      <motion.div
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        className="bg-forge-card border border-forge-border rounded-xl p-4"
      >
        <div className="flex items-center justify-between mb-2">
          <div className="flex items-center gap-2">
            <Lock className="w-4 h-4 text-forge-accent" />
            <span className="text-sm font-semibold text-forge-text">Privacy Score</span>
          </div>
          <span className="text-sm font-bold text-forge-accent">
            {tweaks.length > 0 ? Math.round((appliedCount / tweaks.length) * 100) : 0}%
          </span>
        </div>
        <div className="h-2 bg-forge-bg rounded-full overflow-hidden">
          <motion.div
            initial={{ width: 0 }}
            animate={{
              width: tweaks.length > 0 ? `${(appliedCount / tweaks.length) * 100}%` : "0%",
            }}
            transition={{ duration: 1 }}
            className="h-full bg-forge-accent rounded-full"
          />
        </div>
        <p className="text-xs text-forge-muted mt-2">
          {appliedCount} of {tweaks.length} protections active
        </p>
      </motion.div>

      {/* Quick Actions */}
      <div className="flex gap-3">
        <motion.button
          whileHover={{ scale: 1.02 }}
          whileTap={{ scale: 0.98 }}
          onClick={applyAll}
          disabled={applyingAll}
          className="flex items-center gap-2 px-5 py-2.5 bg-forge-accent/10 border border-forge-accent/30 text-forge-accent rounded-lg text-sm font-semibold hover:bg-forge-accent/20 transition-colors disabled:opacity-50"
        >
          {applyingAll ? <Loader2 className="w-4 h-4 animate-spin" /> : <Shield className="w-4 h-4" />}
          Protect All
        </motion.button>
        <motion.button
          whileHover={{ scale: 1.02 }}
          whileTap={{ scale: 0.98 }}
          onClick={restoreAll}
          disabled={applyingAll}
          className="flex items-center gap-2 px-5 py-2.5 bg-forge-card border border-forge-border text-forge-muted rounded-lg text-sm font-medium hover:text-forge-text transition-colors disabled:opacity-50"
        >
          Restore Defaults
        </motion.button>
      </div>

      {/* Tweaks by Category */}
      {loading ? (
        <div className="flex items-center justify-center py-16">
          <Loader2 className="w-6 h-6 text-forge-accent animate-spin" />
        </div>
      ) : (
        <div className="space-y-5">
          {categories.map((cat) => {
            const CatIcon = categoryIcons[cat] || Eye;
            const catTweaks = tweaks.filter((t) => t.category === cat);
            return (
              <div key={cat}>
                <h3 className="text-xs font-semibold text-forge-muted uppercase tracking-wider mb-2 flex items-center gap-2">
                  <CatIcon className="w-3.5 h-3.5" />
                  {categoryLabels[cat] || cat}
                </h3>
                <div className="space-y-1.5">
                  {catTweaks.map((tweak, i) => (
                    <motion.div
                      key={tweak.id}
                      initial={{ opacity: 0, x: -10 }}
                      animate={{ opacity: 1, x: 0 }}
                      transition={{ delay: i * 0.03 }}
                      className="flex items-center gap-3 p-3 bg-forge-card border border-forge-border rounded-lg"
                    >
                      <button
                        onClick={() => toggleTweak(tweak)}
                        disabled={applying !== null}
                        className={`w-10 h-5 rounded-full transition-colors relative shrink-0 ${
                          tweak.applied ? "bg-forge-accent" : "bg-forge-border"
                        }`}
                      >
                        <motion.div
                          animate={{ x: tweak.applied ? 20 : 2 }}
                          className="absolute top-0.5 w-4 h-4 bg-white rounded-full shadow"
                        />
                      </button>
                      <div className="flex-1">
                        <p className="text-sm font-medium text-forge-text">{tweak.name}</p>
                        <p className="text-xs text-forge-muted">{tweak.description}</p>
                      </div>
                      {applying === tweak.id && (
                        <Loader2 className="w-4 h-4 text-forge-accent animate-spin shrink-0" />
                      )}
                      {tweak.applied && applying !== tweak.id && (
                        <CheckCircle2 className="w-4 h-4 text-forge-accent shrink-0" />
                      )}
                    </motion.div>
                  ))}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
