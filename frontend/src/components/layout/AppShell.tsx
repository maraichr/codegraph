import { Outlet } from "react-router";
import { Breadcrumbs } from "./Breadcrumbs";
import { Sidebar } from "./Sidebar";
import { ToastContainer } from "../ui/ToastContainer";

export function AppShell() {
  return (
    <div className="flex h-screen bg-gray-50">
      <Sidebar />
      <main className="flex-1 overflow-auto p-6">
        <Breadcrumbs />
        <Outlet />
      </main>
      <ToastContainer />
    </div>
  );
}
