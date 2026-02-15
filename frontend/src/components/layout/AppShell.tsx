import { Outlet, useMatch } from "react-router";
import { ToastContainer } from "../ui/ToastContainer";
import { Breadcrumbs } from "./Breadcrumbs";
import { ProjectTabs } from "./ProjectTabs";
import { Sidebar } from "./Sidebar";

export function AppShell() {
  const isProjectRoute = useMatch("/projects/:slug/*");

  return (
    <div className="flex h-screen bg-background">
      <Sidebar />
      <main className="flex-1 overflow-auto p-6">
        <Breadcrumbs />
        {isProjectRoute && <ProjectTabs />}
        <Outlet />
      </main>
      <ToastContainer />
    </div>
  );
}
