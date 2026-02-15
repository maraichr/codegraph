import { type ReactNode, useEffect } from "react";
import { AuthProvider, useAuth } from "react-oidc-context";
import { oidcConfig, isAuthEnabled } from "./config";
import { TokenSync } from "./TokenSync";

function AuthGuard({ children }: { children: ReactNode }) {
  const auth = useAuth();

  useEffect(() => {
    if (!auth.isLoading && !auth.isAuthenticated && !auth.activeNavigator) {
      auth.signinRedirect();
    }
  }, [auth.isLoading, auth.isAuthenticated, auth.activeNavigator]);

  useEffect(() => {
    const handler = () => auth.signinRedirect();
    window.addEventListener("auth:unauthorized", handler);
    return () => window.removeEventListener("auth:unauthorized", handler);
  }, [auth]);

  if (auth.isLoading) {
    return (
      <div className="flex h-screen items-center justify-center">
        <div className="text-muted-foreground">Authenticating…</div>
      </div>
    );
  }

  if (auth.error) {
    return (
      <div className="flex h-screen flex-col items-center justify-center gap-4">
        <p className="text-destructive">
          Authentication error: {auth.error.message}
        </p>
        <button
          onClick={() => auth.signinRedirect()}
          className="rounded-md bg-primary px-4 py-2 text-sm text-primary-foreground"
        >
          Retry Login
        </button>
      </div>
    );
  }

  if (!auth.isAuthenticated) {
    return null;
  }

  return (
    <>
      <TokenSync />
      {children}
    </>
  );
}

export function AuthGate({ children }: { children: ReactNode }) {
  if (!isAuthEnabled) {
    return <>{children}</>;
  }

  return (
    <AuthProvider {...oidcConfig}>
      <AuthGuard>{children}</AuthGuard>
    </AuthProvider>
  );
}
