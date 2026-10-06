"use client";

import { useState } from "react";
import { useSettings } from "../app/useSettings";
import { getDefaultApiBaseUrl } from "../app/settingsStorage";
import { Button } from "../components/ui/Button";

export function SettingsPage() {
  const { settings, setApiKey, setApiBaseUrl, hasApiKey } = useSettings();
  const [draftKey, setDraftKey] = useState(settings.apiKey);
  const [draftBase, setDraftBase] = useState(settings.apiBaseUrl);
  const [saved, setSaved] = useState(false);

  function save() {
    setApiKey(draftKey);
    setApiBaseUrl(draftBase);
    setSaved(true);
    window.setTimeout(() => setSaved(false), 2500);
  }

  function clearKey() {
    setDraftKey("");
    setApiKey("");
    setSaved(true);
    window.setTimeout(() => setSaved(false), 2500);
  }

  return (
    <div className="mx-auto max-w-xl space-y-6">
      <header>
        <h1 className="text-2xl font-semibold text-white md:text-3xl">Settings</h1>
        <p className="mt-2 text-slate-400">
          The activity feed is public. Mutating API routes need a workspace bearer key, stored in
          this browser only.
        </p>
      </header>

      <form
        className="space-y-5 rounded-2xl border border-slate-800 bg-slate-900/30 p-4 md:p-6"
        onSubmit={(event) => {
          event.preventDefault();
          save();
        }}
      >
        <div className="flex flex-col gap-1.5">
          <label htmlFor="api-base" className="text-sm font-medium text-slate-300">
            API base URL
          </label>
          <input
            id="api-base"
            type="url"
            className="rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-sm"
            value={draftBase}
            onChange={(event) => setDraftBase(event.target.value)}
            placeholder={getDefaultApiBaseUrl()}
          />
          <p className="text-xs text-slate-500">
            Defaults to <code className="text-slate-400">{getDefaultApiBaseUrl()}</code> or{" "}
            <code className="text-slate-400">VITE_API_BASE_URL</code> at build time.
          </p>
        </div>

        <div className="flex flex-col gap-1.5">
          <label htmlFor="api-key" className="text-sm font-medium text-slate-300">
            Workspace API key
          </label>
          <input
            id="api-key"
            type="password"
            autoComplete="off"
            className="rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-sm"
            value={draftKey}
            onChange={(event) => setDraftKey(event.target.value)}
            placeholder="Bearer token for /v1/connections…"
          />
          <p className="text-xs text-slate-500">
            Local dev: use <code className="text-slate-400">BOOTSTRAP_API_KEY</code> from{" "}
            <code className="text-slate-400">.env.example</code>.
          </p>
        </div>

        <div className="flex flex-wrap gap-2">
          <Button type="submit">Save</Button>
          <Button type="button" variant="secondary" onClick={clearKey}>
            Clear key
          </Button>
        </div>

        {saved && (
          <p className="text-sm text-emerald-400" role="status">Settings saved.</p>
        )}
        {hasApiKey && !saved && (
          <p className="text-sm text-slate-500">A key is configured for authenticated requests.</p>
        )}
      </form>
    </div>
  );
}
