import { ChevronUp, ExternalLink, FolderKanban, LogOut, Search, Shield } from "lucide-react";
import { useAuth } from "react-oidc-context";
import { NavLink } from "react-router";
import { isAuthEnabled } from "../../auth/config";
import { Avatar, AvatarFallback } from "../ui/avatar";
import { Badge } from "../ui/badge";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "../ui/dropdown-menu";

const navItems = [
  { to: "/", label: "Projects", icon: FolderKanban },
  { to: "/search", label: "Search", icon: Search },
];

function LatticeLogo() {
  return (
    <svg
      width="28"
      height="28"
      viewBox="0 0 28 28"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      role="img"
      aria-label="Lattice logo"
    >
      <title>Lattice</title>
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

function getInitials(name: string): string {
  return name
    .split(" ")
    .map((part) => part[0])
    .filter(Boolean)
    .slice(0, 2)
    .join("")
    .toUpperCase();
}

function AuthenticatedFooter() {
  const auth = useAuth();
  const name = auth.user?.profile?.name || auth.user?.profile?.preferred_username || "User";
  const email = auth.user?.profile?.email;

  const profile = auth.user?.profile as Record<string, unknown> | undefined;
  const realmAccess = profile?.realm_access as { roles?: string[] } | undefined;
  const roles = realmAccess?.roles?.filter(
    (r) => !r.startsWith("default-roles-") && r !== "offline_access" && r !== "uma_authorization",
  ) ?? [];
  const isAdmin = roles.includes("lattice_admin");

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          className="flex w-full items-center gap-2 rounded-md p-1 text-left hover:bg-accent transition-colors"
        >
          <Avatar className="h-8 w-8">
            <AvatarFallback>{getInitials(name)}</AvatarFallback>
          </Avatar>
          <div className="min-w-0 flex-1">
            <p className="truncate text-sm font-medium text-foreground">{name}</p>
          </div>
          <ChevronUp className="h-4 w-4 shrink-0 text-muted-foreground" />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent side="top" align="start" className="w-56">
        <DropdownMenuLabel className="font-normal">
          <div className="flex flex-col gap-1">
            <p className="text-sm font-medium">{name}</p>
            {email && <p className="text-xs text-muted-foreground">{email}</p>}
            {roles.length > 0 && (
              <div className="flex flex-wrap gap-1 pt-1">
                {roles.map((role) => (
                  <Badge key={role} variant="secondary" className="text-[10px] px-1.5 py-0">
                    {role.replace(/_/g, " ")}
                  </Badge>
                ))}
              </div>
            )}
          </div>
        </DropdownMenuLabel>
        {isAdmin && (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuItem asChild>
              <a
                href="http://localhost:8081/admin/lattice/console/"
                target="_blank"
                rel="noopener noreferrer"
              >
                <Shield className="h-4 w-4" />
                Admin Console
                <ExternalLink className="ml-auto h-3 w-3 text-muted-foreground" />
              </a>
            </DropdownMenuItem>
          </>
        )}
        <DropdownMenuSeparator />
        <DropdownMenuItem onClick={() => auth.signoutRedirect()}>
          <LogOut className="h-4 w-4" />
          Sign out
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
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
        <LatticeLogo />
        <h1 className="text-lg font-semibold text-foreground tracking-tight">Lattice</h1>
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
