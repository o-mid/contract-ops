import { createContext } from "react";
import type { ConsoleSettings } from "./settingsTypes";

export type SettingsContextValue = {
  settings: ConsoleSettings;
  setApiKey: (key: string) => void;
  setApiBaseUrl: (url: string) => void;
  hasApiKey: boolean;
};

export const SettingsContext = createContext<SettingsContextValue | null>(null);
