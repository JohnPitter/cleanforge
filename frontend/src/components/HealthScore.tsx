import { motion } from "framer-motion";

interface HealthScoreProps {
  score: number;
}

export default function HealthScore({ score }: HealthScoreProps) {
  const circumference = 2 * Math.PI * 58;
  const strokeDashoffset = circumference - (score / 100) * circumference;

  const color =
    score >= 80
      ? "text-forge-accent"
      : score >= 50
        ? "text-forge-warning"
        : "text-forge-danger";

  const strokeColor =
    score >= 80 ? "#00ff88" : score >= 50 ? "#ffaa00" : "#ff4444";

  const label =
    score >= 80
      ? "Excellent"
      : score >= 60
        ? "Good"
        : score >= 40
          ? "Fair"
          : "Needs Work";

  return (
    <div className="flex flex-col items-center justify-center">
      <div className="relative w-36 h-36">
        <svg className="w-full h-full -rotate-90" viewBox="0 0 128 128">
          <circle
            cx="64"
            cy="64"
            r="58"
            stroke="#2a2a3e"
            strokeWidth="6"
            fill="none"
          />
          <motion.circle
            cx="64"
            cy="64"
            r="58"
            stroke={strokeColor}
            strokeWidth="6"
            fill="none"
            strokeLinecap="round"
            strokeDasharray={circumference}
            initial={{ strokeDashoffset: circumference }}
            animate={{ strokeDashoffset }}
            transition={{ duration: 1.5, ease: "easeOut" }}
            style={{ filter: `drop-shadow(0 0 6px ${strokeColor}40)` }}
          />
        </svg>
        <div className="absolute inset-0 flex flex-col items-center justify-center">
          <motion.span
            className={`text-3xl font-bold ${color}`}
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ delay: 0.5 }}
          >
            {score}
          </motion.span>
          <span className="text-[10px] text-forge-muted uppercase tracking-wider">
            Score
          </span>
        </div>
      </div>
      <p className={`mt-2 text-sm font-medium ${color}`}>{label}</p>
    </div>
  );
}
