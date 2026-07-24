import { apiBaseURL } from "../config";
import type { ApiEnvelope, AuthResult } from "./types";

export class ApiError extends Error {
  constructor(public readonly code: string, message: string, public readonly status?: number) { super(message); }
}

let accessToken: string | null = null;
let refreshInFlight: Promise<AuthResult> | null = null;

export const sessionToken = { get: () => accessToken, set: (token: string | null) => { accessToken = token; } };

type RequestOptions = Omit<RequestInit, "body" | "headers"> & { body?: unknown; headers?: HeadersInit; auth?: boolean; retry?: boolean; idempotencyKey?: string };

async function decode<T>(response: Response): Promise<T> {
  let payload: ApiEnvelope<T>;
  try { payload = await response.json() as ApiEnvelope<T>; } catch { throw new ApiError("INVALID_RESPONSE", "API returned an invalid JSON response", response.status); }
  if (!response.ok) throw new ApiError(payload.error?.code ?? "HTTP_ERROR", payload.error?.message ?? `HTTP ${response.status}`, response.status);
  if (!payload.success) throw new ApiError(payload.error?.code ?? "UNKNOWN", payload.error?.message ?? "API request failed", response.status);
  return payload.data;
}

export async function refreshSession(): Promise<AuthResult> {
  if (!refreshInFlight) {
    refreshInFlight = request<AuthResult>("/api/v1/auth/refresh", { method: "POST", auth: false, retry: false })
      .then((result) => { sessionToken.set(result.access_token); return result; })
      .finally(() => { refreshInFlight = null; });
  }
  return refreshInFlight;
}

export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { body, auth = true, retry = true, idempotencyKey, headers: providedHeaders, ...init } = options;
  const headers = new Headers(providedHeaders);
  if (body !== undefined) headers.set("Content-Type", "application/json");
  if (auth && accessToken) headers.set("Authorization", `Bearer ${accessToken}`);
  if (idempotencyKey) { headers.set("Idempotency-Key", idempotencyKey); headers.set("X-Correlation-ID", idempotencyKey); }
  let response: Response;
  try {
    response = await fetch(`${apiBaseURL()}${path}`, { ...init, headers, body: body === undefined ? undefined : JSON.stringify(body), credentials: "include" });
  } catch { throw new ApiError("NETWORK_ERROR", "Unable to reach the API"); }
  if (response.status === 401 && auth && retry) {
    try { await refreshSession(); } catch { sessionToken.set(null); throw new ApiError("SESSION_EXPIRED", "Your session has expired", 401); }
    return request<T>(path, { ...options, retry: false });
  }
  return decode<T>(response);
}

export function write<T>(method: "POST" | "PUT" | "PATCH" | "DELETE", path: string, body?: unknown): Promise<T> {
  return request<T>(path, { method, body, idempotencyKey: crypto.randomUUID() });
}
