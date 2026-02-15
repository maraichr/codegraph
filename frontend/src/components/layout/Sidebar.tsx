import { FolderKanban, Search } from "lucide-react";
import { NavLink } from "react-router";

const navItems = [
  { to: "/", label: "Projects", icon: FolderKanban },
  { to: "/search", label: "Search", icon: Search },
];

export function Sidebar() {
  return (
    <aside className="flex w-64 flex-col border-r border-border bg-background">
      <div className="flex h-16 items-center border-b border-border px-6">
        <h1 className="text-xl font-bold text-foreground">CodeGraph</h1>
      </div>
      <nav className="flex-1 space-y-1 p-4">
        {navItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end
            className={({ isActive }) =>
              `flex items-center gap-2 rounded-md px-3 py-2 text-sm font-medium ${
                isActive
                  ? "bg-accent text-accent-foreground"
                  : "text-muted-foreground hover:bg-accent hover:text-accent-foreground"
              }`
            }
          >
            <item.icon className="h-4 w-4" />
            {item.label}
          </NavLink>
        ))}
      </nav>
      <div className="border-t border-border p-4">
        <p className="text-xs text-muted-foreground">CodeGraph v1.0</p>
      </div>
    </aside>
  );
}
