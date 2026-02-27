import { ReactNode } from "react";
import { motion } from "framer-motion";
import { LucideIcon } from "lucide-react";

interface StatCardProps {
  icon: LucideIcon;
  label: string;
  value: string;
  subValue?: string;
  color?: string;
  percentage?: number;
  children?: ReactNode;
}

export default function StatCard({
  icon: Icon,
  label,
  value,
  subValue,
  color = "text-forge-accent",
  percentage,
  children,
}: StatCardProps) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      className="bg-forge-card border border-forge-border rounded-xl p-4 hover:border-forge-accent/30 transition-colors"
    >
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-2 mb-3">
          <Icon className={`w-4 h-4 ${color}`} />
          <span className="text-xs text-forge-muted uppercase tracking-wider">
            {label}
          </span>
        </div>
        {percentage !== undefined && (
          <span
            className={`text-xs font-mono ${
              percentage > 80
                ? "text-forge-danger"
                : percentage > 60
                  ? "text-forge-warning"
                  : "text-forge-accent"
            }`}
          >
            {percentage.toFixed(0)}%
          </span>
        )}
      </div>
      <p className={`text-2xl font-bold ${color}`}>{value}</p>
      {subValue && (
        <p className="text-xs text-forge-muted mt-1">{subValue}</p>
      )}
      {percentage !== undefined && (
        <div className="mt-3 h-1.5 bg-forge-bg rounded-full overflow-hidden">
          <motion.div
            initial={{ width: 0 }}
            animate={{ width: `${percentage}%` }}
            transition={{ duration: 1, ease: "easeOut" }}
            className={`h-full rounded-full ${
              percentage > 80
                ? "bg-forge-danger"
                : percentage > 60
                  ? "bg-forge-warning"
                  : "bg-forge-accent"
            }`}
          />
        </div>
      )}
      {children && <div className="mt-3 border-t border-forge-border pt-3">{children}</div>}
    </motion.div>
  );
}
