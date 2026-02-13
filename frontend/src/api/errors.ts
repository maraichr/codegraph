/** Machine-readable error code from the backend. */
export class ApiError extends Error {
  readonly code: string;
  readonly status: number;

  constructor(code: string, message: string, status: number) {
    super(message);
    this.name = "ApiError";
    this.code = code;
    this.status = status;
  }

  get isClientError(): boolean {
    return this.status >= 400 && this.status < 500;
  }

  get isServerError(): boolean {
    return this.status >= 500;
  }

  get isNotFound(): boolean {
    return this.status === 404;
  }
}

/**
 * Parse an error response from the API into an ApiError.
 * Handles both the new structured format and the legacy string format.
 */
export async function parseApiError(res: Response): Promise<ApiError> {
  try {
    const body = await res.json();
    // New format: { error: { code: "...", message: "..." } }
    if (body?.error?.code) {
      return new ApiError(body.error.code, body.error.message, res.status);
    }
    // Legacy format: { error: "string" }
    if (typeof body?.error === "string") {
      return new ApiError("UNKNOWN", body.error, res.status);
    }
  } catch {
    // JSON parse failed — fall through
  }
  return new ApiError("UNKNOWN", `API error: ${res.status} ${res.statusText}`, res.status);
}
