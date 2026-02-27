import { Routes, Route } from "react-router-dom";
import {
  WindowMinimise,
  WindowToggleMaximise,
  Quit,
} from "../wailsjs/runtime/runtime";
import Sidebar from "./components/Sidebar";
import Dashboard from "./pages/Dashboard";
import Cleaner from "./pages/Cleaner";
import GameBoost from "./pages/GameBoost";
import Startup from "./pages/Startup";
import Network from "./pages/Network";
import Toolkit from "./pages/Toolkit";
import Privacy from "./pages/Privacy";
import Settings from "./pages/Settings";

function App() {
  return (
    <div className="flex flex-col h-screen bg-forge-bg overflow-hidden">
      {/* Title bar — spans full window width */}
      <div
        className="h-8 w-full bg-forge-surface border-b border-forge-border flex items-center shrink-0"
        style={{ WebkitAppRegion: "drag" } as any}
      >
        <div className="flex-1 h-full" />
        <div
          className="flex items-center h-full"
          style={{ WebkitAppRegion: "no-drag" } as any}
        >
          <button
            onClick={WindowMinimise}
            className="h-full px-3.5 text-forge-muted hover:text-forge-text hover:bg-forge-card transition-colors"
            title="Minimize"
          >
            <svg width="10" height="1" viewBox="0 0 10 1" fill="currentColor">
              <rect width="10" height="1" />
            </svg>
          </button>
          <button
            onClick={WindowToggleMaximise}
            className="h-full px-3.5 text-forge-muted hover:text-forge-text hover:bg-forge-card transition-colors"
            title="Maximize"
          >
            <svg width="10" height="10" viewBox="0 0 10 10" fill="none" stroke="currentColor" strokeWidth="1">
              <rect x="0.5" y="0.5" width="9" height="9" />
            </svg>
          </button>
          <button
            onClick={Quit}
            className="h-full px-3.5 text-forge-muted hover:text-white hover:bg-red-600 transition-colors"
            title="Close"
          >
            <svg width="10" height="10" viewBox="0 0 10 10" fill="none" stroke="currentColor" strokeWidth="1.2">
              <line x1="0" y1="0" x2="10" y2="10" />
              <line x1="10" y1="0" x2="0" y2="10" />
            </svg>
          </button>
        </div>
      </div>
      {/* Main content area */}
      <div className="flex flex-1 overflow-hidden">
        <Sidebar />
        <main className="flex-1 overflow-hidden">
          <div className="h-full">
            <Routes>
              <Route path="/" element={<Dashboard />} />
              <Route path="/cleaner" element={<Cleaner />} />
              <Route path="/gameboost" element={<GameBoost />} />
              <Route path="/startup" element={<Startup />} />
              <Route path="/network" element={<Network />} />
              <Route path="/toolkit" element={<Toolkit />} />
              <Route path="/privacy" element={<Privacy />} />
              <Route path="/settings" element={<Settings />} />
            </Routes>
          </div>
        </main>
      </div>
    </div>
  );
}

export default App;
