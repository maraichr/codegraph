import type { AuthProviderProps } from "react-oidc-context";

export const oidcConfig: AuthProviderProps = {
  authority:
    import.meta.env.VITE_AUTH_AUTHORITY ??
    "http://localhost:8081/realms/codegraph",
  client_id: import.meta.env.VITE_AUTH_CLIENT_ID ?? "codegraph-public",
  redirect_uri: window.location.origin + "/",
  post_logout_redirect_uri: window.location.origin + "/",
  scope: "openid",
  automaticSilentRenew: true,
  onSigninCallback: () => {
    // Remove OIDC query params from URL after login
    window.history.replaceState({}, document.title, window.location.pathname);
  },
};

export const isAuthEnabled =
  (import.meta.env.VITE_AUTH_ENABLED ?? "false") === "true";
