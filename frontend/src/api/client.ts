import { parseApiError } from "./errors";

const BASE_URL = import.meta.env.VITE_API_BASE_URL || "";

class ApiClient {
  private baseUrl: string;
  private tokenProvider?: () => Promise<string | undefined>;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }

  setTokenProvider(provider: () => Promise<string | undefined>) {
    this.tokenProvider = provider;
  }

  private async authHeaders(): Promise<Record<string, string>> {
    const token = await this.tokenProvider?.();
    return token ? { Authorization: `Bearer ${token}` } : {};
  }

  private handleUnauthorized(res: Response) {
    if (res.status === 401) {
      window.dispatchEvent(new Event("auth:unauthorized"));
    }
  }

  async get<T>(path: string): Promise<T> {
    const headers = await this.authHeaders();
    const res = await fetch(`${this.baseUrl}${path}`, { headers });
    if (!res.ok) {
      this.handleUnauthorized(res);
      throw await parseApiError(res);
    }
    return res.json() as Promise<T>;
  }

  async post<T>(path: string, body: unknown): Promise<T> {
    const headers = await this.authHeaders();
    const res = await fetch(`${this.baseUrl}${path}`, {
      method: "POST",
      headers: { "Content-Type": "application/json", ...headers },
      body: JSON.stringify(body),
    });
    if (!res.ok) {
      this.handleUnauthorized(res);
      throw await parseApiError(res);
    }
    return res.json() as Promise<T>;
  }

  async put<T>(path: string, body: unknown): Promise<T> {
    const headers = await this.authHeaders();
    const res = await fetch(`${this.baseUrl}${path}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json", ...headers },
      body: JSON.stringify(body),
    });
    if (!res.ok) {
      this.handleUnauthorized(res);
      throw await parseApiError(res);
    }
    return res.json() as Promise<T>;
  }

  async delete(path: string): Promise<void> {
    const headers = await this.authHeaders();
    const res = await fetch(`${this.baseUrl}${path}`, {
      method: "DELETE",
      headers,
    });
    if (!res.ok) {
      this.handleUnauthorized(res);
      throw await parseApiError(res);
    }
  }

  async upload<T>(path: string, file: File): Promise<T> {
    const headers = await this.authHeaders();
    const formData = new FormData();
    formData.append("file", file);
    const res = await fetch(`${this.baseUrl}${path}`, {
      method: "POST",
      headers,
      body: formData,
    });
    if (!res.ok) {
      this.handleUnauthorized(res);
      throw await parseApiError(res);
    }
    return res.json() as Promise<T>;
  }
}

export const apiClient = new ApiClient(BASE_URL);
