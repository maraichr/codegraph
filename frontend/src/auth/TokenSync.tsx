import { useEffect } from "react";
import { useAuth } from "react-oidc-context";
import { apiClient } from "../api/client";

export function TokenSync() {
  const auth = useAuth();

  useEffect(() => {
    apiClient.setTokenProvider(async () => auth.user?.access_token);
  }, [auth.user?.access_token]);

  return null;
}
