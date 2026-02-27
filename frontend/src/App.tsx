import { Routes, Route } from "react-router-dom";
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
    <div className="flex h-screen bg-forge-bg overflow-hidden">
      <Sidebar />
      <main className="flex-1 overflow-hidden">
        <div
          className="h-8 w-full bg-forge-surface border-b border-forge-border"
          style={{ WebkitAppRegion: "drag" } as any}
        />
        <div className="h-[calc(100vh-2rem)]">
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
  );
}

export default App;
