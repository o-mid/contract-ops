export const API_KEY_STORAGE = "contract_ops_api_key";
export const API_BASE_STORAGE = "contract_ops_api_base";

const PRODUCTION_API = "https://api-production-4b82.up.railway.app";

function inferApiBaseFromHost(): string {
  if (typeof window === "undefined") {
    return "http://localhost:8080";
  }
  const host = window.location.hostname;
  if (host === "localhost" || host === "127.0.0.1") {
    return "http://localhost:8080";
  }
  if (host.endsWith(".up.railway.app")) {
    return PRODUCTION_API;
  }
  return PRODUCTION_API;
}

export const defaultApiBaseUrl =
  process.env.NEXT_PUBLIC_API_BASE_URL?.trim() || inferApiBaseFromHost();

export function readStorage(key: string): string {
  try {
    return localStorage.getItem(key) ?? "";
  } catch {
    return "";
  }
}

export function writeStorage(key: string, value: string) {
  try {
    if (value) {
      localStorage.setItem(key, value);
    } else {
      localStorage.removeItem(key);
    }
  } catch {
    /* private mode */
  }
}

export function getDefaultApiBaseUrl() {
  return defaultApiBaseUrl;
}

/** Ignore a stale local API URL when the console runs on Railway but storage still has localhost. */
export function resolveApiBaseUrl(stored: string): string {
  const trimmed = stored.trim();
  if (!trimmed) {
    return defaultApiBaseUrl;
  }
  if (
    typeof window !== "undefined" &&
    window.location.hostname.endsWith(".up.railway.app") &&
    (trimmed.includes("localhost") || trimmed.includes("127.0.0.1"))
  ) {
    return defaultApiBaseUrl;
  }
  return trimmed;
}
