import { BarChart3, GitFork, LayoutDashboard, Zap } from "lucide-react";
import { NavLink, useParams } from "react-router";
import { cn } from "@/lib/utils";

const tabs = [
  { to: "", label: "Overview", icon: LayoutDashboard, end: true },
  { to: "/graph", label: "Graph", icon: GitFork, end: false },
  { to: "/lineage", label: "Lineage", icon: BarChart3, end: false },
  { to: "/impact", label: "Impact", icon: Zap, end: false },
];

export function ProjectTabs() {
  const { slug } = useParams<{ slug: string }>();
  if (!slug) return null;

  const base = `/projects/${slug}`;

  return (
    <div className="mb-6 flex gap-1 rounded-lg bg-muted p-1">
      {tabs.map((tab) => (
        <NavLink
          key={tab.to}
          to={`${base}${tab.to}`}
          end={tab.end}
          className={({ isActive }) =>
            cn(
              "inline-flex items-center gap-2 rounded-md px-3 py-1.5 text-sm font-medium transition-all",
              isActive
                ? "bg-background text-foreground shadow"
                : "text-muted-foreground hover:text-foreground",
            )
          }
        >
          <tab.icon className="h-4 w-4" />
          {tab.label}
        </NavLink>
      ))}
    </div>
  );
}
