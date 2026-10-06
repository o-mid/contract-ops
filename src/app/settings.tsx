import { useCallback, useMemo, useState, type ReactNode } from "react";
import { SettingsContext } from "./settingsContext";
import {
  API_BASE_STORAGE,
  API_KEY_STORAGE,
  defaultApiBaseUrl,
  readStorage,
  writeStorage
} from "./settingsStorage";

export function SettingsProvider({ children }: { children: ReactNode }) {
  const [apiKey, setApiKeyState] = useState(() => readStorage(API_KEY_STORAGE));
  const [apiBaseUrl, setApiBaseUrlState] = useState(() => {
    const stored = readStorage(API_BASE_STORAGE);
    return stored || defaultApiBaseUrl;
  });

  const setApiKey = useCallback((key: string) => {
    setApiKeyState(key);
    writeStorage(API_KEY_STORAGE, key);
  }, []);

  const setApiBaseUrl = useCallback((url: string) => {
    const trimmed = url.trim();
    setApiBaseUrlState(trimmed || defaultApiBaseUrl);
    writeStorage(API_BASE_STORAGE, trimmed);
  }, []);

  const value = useMemo(
    () => ({
      settings: { apiKey, apiBaseUrl: apiBaseUrl || defaultApiBaseUrl },
      setApiKey,
      setApiBaseUrl,
      hasApiKey: apiKey.trim().length > 0
    }),
    [apiKey, apiBaseUrl, setApiKey, setApiBaseUrl]
  );

  return <SettingsContext.Provider value={value}>{children}</SettingsContext.Provider>;
}
