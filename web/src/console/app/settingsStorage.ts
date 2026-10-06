export const API_KEY_STORAGE = "contract_ops_api_key";
export const API_BASE_STORAGE = "contract_ops_api_base";

export const defaultApiBaseUrl =
  process.env.NEXT_PUBLIC_API_BASE_URL?.trim() || "http://localhost:8080";

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
