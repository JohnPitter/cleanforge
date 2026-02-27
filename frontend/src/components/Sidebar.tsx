import { useState, useEffect } from "react";
import { NavLink } from "react-router-dom";
import { motion } from "framer-motion";
import {
  LayoutDashboard,
  Trash2,
  Gamepad2,
  Rocket,
  Wifi,
  Wrench,
  ShieldCheck,
  Settings,
  Flame,
} from "lucide-react";

const navItems = [
  { to: "/", icon: LayoutDashboard, label: "Dashboard" },
  { to: "/cleaner", icon: Trash2, label: "Cleaner" },
  { to: "/gameboost", icon: Gamepad2, label: "Game Boost" },
  { to: "/startup", icon: Rocket, label: "Startup" },
  { to: "/network", icon: Wifi, label: "Network" },
  { to: "/toolkit", icon: Wrench, label: "Toolkit" },
  { to: "/privacy", icon: ShieldCheck, label: "Privacy" },
  { to: "/settings", icon: Settings, label: "Settings" },
];

export default function Sidebar() {
  const [version, setVersion] = useState("");

  useEffect(() => {
    // @ts-ignore - Wails bindings
    window.go?.main?.App?.GetVersion?.().then((v: string) => setVersion(v));
  }, []);

  return (
    <aside className="w-56 h-screen bg-forge-surface border-r border-forge-border flex flex-col shrink-0">
      <div className="p-5 border-b border-forge-border">
        <div className="flex items-center gap-2">
          <Flame className="w-7 h-7 text-forge-accent" />
          <div>
            <h1 className="text-lg font-bold text-forge-accent tracking-tight">
              CLEAN
              <span className="text-forge-text">FORGE</span>
            </h1>
            <p className="text-[10px] text-forge-muted tracking-widest uppercase">
              Performance Suite
            </p>
          </div>
        </div>
      </div>

      <nav className="flex-1 py-3 px-2 space-y-0.5 overflow-y-auto">
        {navItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.to === "/"}
            className={({ isActive }) =>
              `flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all duration-200 ${
                isActive
                  ? "bg-forge-accent/10 text-forge-accent border border-forge-accent/20"
                  : "text-forge-muted hover:text-forge-text hover:bg-forge-card border border-transparent"
              }`
            }
          >
            {({ isActive }) => (
              <>
                <item.icon
                  className={`w-4.5 h-4.5 ${isActive ? "text-forge-accent" : ""}`}
                />
                <span>{item.label}</span>
                {isActive && (
                  <motion.div
                    layoutId="sidebar-indicator"
                    className="ml-auto w-1.5 h-1.5 rounded-full bg-forge-accent"
                    transition={{ type: "spring", stiffness: 350, damping: 30 }}
                  />
                )}
              </>
            )}
          </NavLink>
        ))}
      </nav>

      <div className="p-4 border-t border-forge-border">
        <p className="text-[10px] text-forge-muted text-center">
          CleanForge {version ? `v${version}` : ""}
        </p>
      </div>
    </aside>
  );
}
