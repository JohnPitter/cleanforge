import { useState } from "react";
import { motion } from "framer-motion";
import {
  Settings as SettingsIcon,
  Info,
  Github,
  Flame,
  Calendar,
  Clock,
  RotateCcw,
} from "lucide-react";

export default function Settings() {
  const [autoClean, setAutoClean] = useState(false);
  const [autoBoost, setAutoBoost] = useState(false);
  const [cleanInterval, setCleanInterval] = useState("weekly");

  return (
    <div className="p-6 space-y-6 overflow-y-auto h-full">
      <div>
        <h2 className="text-2xl font-bold text-forge-text flex items-center gap-2">
          <SettingsIcon className="w-6 h-6 text-forge-accent" /> Settings
        </h2>
        <p className="text-sm text-forge-muted">Configure CleanForge behavior</p>
      </div>

      {/* Automation */}
      <div className="bg-forge-card border border-forge-border rounded-xl p-5 space-y-4">
        <h3 className="text-sm font-semibold text-forge-text flex items-center gap-2">
          <Calendar className="w-4 h-4 text-forge-accent" /> Automation
        </h3>

        <div className="flex items-center justify-between">
          <div>
            <p className="text-sm text-forge-text">Scheduled Cleanup</p>
            <p className="text-xs text-forge-muted">Automatically clean temp files on a schedule</p>
          </div>
          <button
            onClick={() => setAutoClean(!autoClean)}
            className={`w-10 h-5 rounded-full transition-colors relative ${
              autoClean ? "bg-forge-accent" : "bg-forge-border"
            }`}
          >
            <motion.div
              animate={{ x: autoClean ? 20 : 2 }}
              className="absolute top-0.5 w-4 h-4 bg-white rounded-full shadow"
            />
          </button>
        </div>

        {autoClean && (
          <motion.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: "auto" }}
            className="flex gap-2 pl-4"
          >
            {["daily", "weekly", "monthly"].map((opt) => (
              <button
                key={opt}
                onClick={() => setCleanInterval(opt)}
                className={`px-3 py-1.5 rounded-md text-xs font-medium transition-colors ${
                  cleanInterval === opt
                    ? "bg-forge-accent/10 text-forge-accent border border-forge-accent/30"
                    : "bg-forge-surface text-forge-muted border border-forge-border"
                }`}
              >
                {opt.charAt(0).toUpperCase() + opt.slice(1)}
              </button>
            ))}
          </motion.div>
        )}

        <div className="flex items-center justify-between">
          <div>
            <p className="text-sm text-forge-text">Auto Game Boost</p>
            <p className="text-xs text-forge-muted">Detect game launches and apply boost automatically</p>
          </div>
          <button
            onClick={() => setAutoBoost(!autoBoost)}
            className={`w-10 h-5 rounded-full transition-colors relative ${
              autoBoost ? "bg-forge-accent" : "bg-forge-border"
            }`}
          >
            <motion.div
              animate={{ x: autoBoost ? 20 : 2 }}
              className="absolute top-0.5 w-4 h-4 bg-white rounded-full shadow"
            />
          </button>
        </div>
      </div>

      {/* About */}
      <div className="bg-forge-card border border-forge-border rounded-xl p-5 space-y-4">
        <h3 className="text-sm font-semibold text-forge-text flex items-center gap-2">
          <Info className="w-4 h-4 text-forge-info" /> About
        </h3>

        <div className="flex items-center gap-4">
          <Flame className="w-10 h-10 text-forge-accent" />
          <div>
            <p className="text-lg font-bold text-forge-accent">
              CLEAN<span className="text-forge-text">FORGE</span>
            </p>
            <p className="text-xs text-forge-muted">v1.0.0 | Open Source Performance Suite</p>
          </div>
        </div>

        <p className="text-xs text-forge-muted leading-relaxed">
          CleanForge is the ultimate Windows performance tool. It combines system cleanup,
          gaming optimization, startup management, network tuning, privacy protection, and
          system repair tools — all in one modern, open-source application.
        </p>

        <div className="flex gap-3">
          <a
            href="https://github.com/JohnPitter/cleanforge"
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-1.5 px-3 py-1.5 bg-forge-surface border border-forge-border rounded-md text-xs text-forge-muted hover:text-forge-text transition-colors"
          >
            <Github className="w-3 h-3" /> GitHub
          </a>
        </div>
      </div>
    </div>
  );
}
