import { request, sessionToken } from "./http";
import type { AuthResult } from "./types";

export async function login(email: string, password: string): Promise<AuthResult> {
  const result = await request<AuthResult>("/api/v1/admin/auth/login", { method: "POST", body: { email, password }, auth: false, retry: false });
  sessionToken.set(result.access_token);
  return result;
}
export async function logout(): Promise<void> {
  try { await request("/api/v1/auth/logout", { method: "POST", auth: false, retry: false }); } finally { sessionToken.set(null); }
}
