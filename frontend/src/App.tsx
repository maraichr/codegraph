import { Route, Routes } from "react-router";
import { AppShell } from "./components/layout/AppShell";
import { GraphExplorer } from "./pages/GraphExplorer";
import { ImpactAnalysis } from "./pages/ImpactAnalysis";
import { LineageExplorer } from "./pages/LineageExplorer";
import { ProjectDashboard } from "./pages/ProjectDashboard";
import { ProjectList } from "./pages/ProjectList";

export default function App() {
  return (
    <Routes>
      <Route element={<AppShell />}>
        <Route index element={<ProjectList />} />
        <Route path="projects" element={<ProjectList />} />
        <Route path="projects/:slug" element={<ProjectDashboard />} />
        <Route path="projects/:slug/graph" element={<GraphExplorer />} />
        <Route path="projects/:slug/lineage" element={<LineageExplorer />} />
        <Route path="projects/:slug/impact" element={<ImpactAnalysis />} />
      </Route>
    </Routes>
  );
}
