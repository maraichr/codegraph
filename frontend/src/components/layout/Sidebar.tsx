import { FolderKanban, LogOut, Search } from "lucide-react";
import { useAuth } from "react-oidc-context";
import { NavLink } from "react-router";
import { isAuthEnabled } from "../../auth/config";

const navItems = [
  { to: "/", label: "Projects", icon: FolderKanban },
  { to: "/search", label: "Search", icon: Search },
];

function CodeGraphLogo() {
  return (
    <svg
      width="28"
      height="28"
      viewBox="0 0 28 28"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      role="img"
      aria-label="CodeGraph logo"
    >
      <title>CodeGraph</title>
      <circle cx="14" cy="6" r="3.5" fill="hsl(185 70% 50%)" />
      <circle cx="6" cy="22" r="3.5" fill="hsl(185 70% 50%)" opacity="0.7" />
      <circle cx="22" cy="22" r="3.5" fill="hsl(185 70% 50%)" opacity="0.5" />
      <line
        x1="14"
        y1="9.5"
        x2="6"
        y2="18.5"
        stroke="hsl(185 70% 50%)"
        strokeWidth="1.5"
        opacity="0.5"
      />
      <line
        x1="14"
        y1="9.5"
        x2="22"
        y2="18.5"
        stroke="hsl(185 70% 50%)"
        strokeWidth="1.5"
        opacity="0.5"
      />
      <line
        x1="6"
        y1="22"
        x2="22"
        y2="22"
        stroke="hsl(185 70% 50%)"
        strokeWidth="1.5"
        opacity="0.3"
      />
    </svg>
  );
}

function AuthenticatedFooter() {
  const auth = useAuth();
  const name = auth.user?.profile?.name || auth.user?.profile?.preferred_username || "User";
  const email = auth.user?.profile?.email;

  return (
    <div className="flex items-center justify-between gap-2">
      <div className="min-w-0">
        <p className="truncate text-sm font-medium text-foreground">{name}</p>
        {email && <p className="truncate text-xs text-muted-foreground">{email}</p>}
      </div>
      <button
        type="button"
        onClick={() => auth.signoutRedirect()}
        title="Sign out"
        className="shrink-0 rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-accent-foreground"
      >
        <LogOut className="h-4 w-4" />
      </button>
    </div>
  );
}

function UserFooter() {
  if (!isAuthEnabled) {
    return <p className="text-xs text-muted-foreground">Dev Mode</p>;
  }
  return <AuthenticatedFooter />;
}

export function Sidebar() {
  return (
    <aside className="flex w-64 flex-col border-r border-border bg-background">
      <div className="flex h-16 items-center gap-3 border-b border-border px-6">
        <CodeGraphLogo />
        <h1 className="text-lg font-semibold text-foreground tracking-tight">CodeGraph</h1>
      </div>
      <nav className="flex-1 space-y-1 p-3">
        {navItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end
            className={({ isActive }) =>
              `flex items-center gap-3 rounded-md px-3 py-2.5 text-sm font-medium transition-colors ${
                isActive
                  ? "border-l-2 border-primary bg-primary/10 text-primary"
                  : "border-l-2 border-transparent text-muted-foreground hover:bg-accent hover:text-accent-foreground"
              }`
            }
          >
            <item.icon className="h-4 w-4" />
            {item.label}
          </NavLink>
        ))}
      </nav>
      <div className="border-t border-border p-4">
        <UserFooter />
      </div>
      <div className="px-4 pb-3">
        <p className="text-[10px] font-mono text-muted-foreground/50">v1.0.0</p>
      </div>
    </aside>
  );
}
