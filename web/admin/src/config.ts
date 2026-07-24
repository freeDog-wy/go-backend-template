export function apiBaseURL(): string {
  const value = window.__ADMIN_CONFIG__?.apiBaseURL?.trim();
  if (!value) throw new Error("Missing runtime API configuration");
  const url = new URL(value);
  if (!["https:", "http:"].includes(url.protocol) || url.search || url.hash) {
    throw new Error("Invalid runtime API URL");
  }
  if (url.protocol !== "https:" && url.hostname !== "localhost") {
    throw new Error("The API URL must use HTTPS outside localhost");
  }
  return url.toString().replace(/\/$/, "");
}
