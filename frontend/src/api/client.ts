import { parseApiError } from "./errors";

const BASE_URL = "";

class ApiClient {
  private baseUrl: string;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }

  async get<T>(path: string): Promise<T> {
    const res = await fetch(`${this.baseUrl}${path}`);
    if (!res.ok) {
      throw await parseApiError(res);
    }
    return res.json() as Promise<T>;
  }

  async post<T>(path: string, body: unknown): Promise<T> {
    const res = await fetch(`${this.baseUrl}${path}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    if (!res.ok) {
      throw await parseApiError(res);
    }
    return res.json() as Promise<T>;
  }

  async put<T>(path: string, body: unknown): Promise<T> {
    const res = await fetch(`${this.baseUrl}${path}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    if (!res.ok) {
      throw await parseApiError(res);
    }
    return res.json() as Promise<T>;
  }

  async delete(path: string): Promise<void> {
    const res = await fetch(`${this.baseUrl}${path}`, {
      method: "DELETE",
    });
    if (!res.ok) {
      throw await parseApiError(res);
    }
  }

  async upload<T>(path: string, file: File): Promise<T> {
    const formData = new FormData();
    formData.append("file", file);
    const res = await fetch(`${this.baseUrl}${path}`, {
      method: "POST",
      body: formData,
    });
    if (!res.ok) {
      throw await parseApiError(res);
    }
    return res.json() as Promise<T>;
  }
}

export const apiClient = new ApiClient(BASE_URL);
