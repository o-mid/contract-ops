import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { ApiError } from "../api/client";
import {
  backfillConnection,
  createConnection,
  listConnections,
  verifyConnection
} from "../api/connections";
import { useSettings } from "../app/useSettings";
import { Button } from "../components/ui/Button";
import type { ConnectionKind } from "../domain/connection";
import { formatTimestamp } from "../lib/format";
import { ConnectionStatusBadge } from "../features/connections/ConnectionStatusBadge";

export function ConnectionsPage() {
  const { settings, hasApiKey } = useSettings();
  const queryClient = useQueryClient();
  const [name, setName] = useState("Demo vendor");
  const [secret, setSecret] = useState("demo-secret");
  const [kind, setKind] = useState<ConnectionKind>("fakevendor");
  const [actionMessage, setActionMessage] = useState<string | null>(null);

  const connectionsQuery = useQuery({
    queryKey: ["connections", settings.apiBaseUrl],
    queryFn: ({ signal }) =>
      listConnections({
        apiBaseUrl: settings.apiBaseUrl,
        apiKey: settings.apiKey,
        signal
      }),
    enabled: hasApiKey
  });

  const createMutation = useMutation({
    mutationFn: () =>
      createConnection(
        { kind, name: name.trim(), secret: secret.trim() },
        { apiBaseUrl: settings.apiBaseUrl, apiKey: settings.apiKey }
      ),
    onSuccess: async () => {
      setActionMessage("Connection created.");
      await queryClient.invalidateQueries({ queryKey: ["connections"] });
    },
    onError: (error: unknown) => {
      setActionMessage(error instanceof ApiError ? error.userMessage() : "Create failed.");
    }
  });

  const verifyMutation = useMutation({
    mutationFn: (id: string) =>
      verifyConnection(id, { apiBaseUrl: settings.apiBaseUrl, apiKey: settings.apiKey }),
    onSuccess: async () => {
      setActionMessage("Connection verified.");
      await queryClient.invalidateQueries({ queryKey: ["connections"] });
    },
    onError: (error: unknown) => {
      setActionMessage(error instanceof ApiError ? error.userMessage() : "Verify failed.");
    }
  });

  const backfillMutation = useMutation({
    mutationFn: (id: string) => {
      const end = new Date();
      const start = new Date(end.getTime() - 24 * 60 * 60 * 1000);
      return backfillConnection(
        id,
        start.toISOString(),
        end.toISOString(),
        { apiBaseUrl: settings.apiBaseUrl, apiKey: settings.apiKey }
      );
    },
    onSuccess: (jobs) => {
      setActionMessage(`Backfill queued (${jobs.length} job${jobs.length === 1 ? "" : "s"}).`);
    },
    onError: (error: unknown) => {
      setActionMessage(error instanceof ApiError ? error.userMessage() : "Backfill failed.");
    }
  });

  if (!hasApiKey) {
    return (
      <div className="max-w-xl space-y-4">
        <h1 className="text-2xl font-semibold text-white">Connections</h1>
        <p className="text-slate-400">
          Add your workspace API key in Settings to list connections, verify credentials, and
          queue backfill jobs against the control plane API.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-8">
      <header className="max-w-3xl">
        <h1 className="text-2xl font-semibold tracking-tight text-white md:text-3xl">
          Connections
        </h1>
        <p className="mt-2 text-slate-400">
          Vendor integrations for this workspace. Secrets are sealed server-side; only fingerprints
          are shown here.
        </p>
      </header>

      {actionMessage && (
        <p className="rounded-lg border border-indigo-500/30 bg-indigo-950/40 px-4 py-3 text-sm text-indigo-100" role="status">
          {actionMessage}
        </p>
      )}

      <section className="rounded-2xl border border-slate-800 bg-slate-900/30 p-4 md:p-6">
        <h2 className="text-lg font-semibold text-white">Add connection</h2>
        <form
          className="mt-4 grid gap-4 md:grid-cols-2"
          onSubmit={(event) => {
            event.preventDefault();
            createMutation.mutate();
          }}
        >
          <div className="flex flex-col gap-1.5">
            <label htmlFor="conn-name" className="text-sm text-slate-300">Name</label>
            <input
              id="conn-name"
              className="rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-sm"
              value={name}
              onChange={(event) => setName(event.target.value)}
              required
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <label htmlFor="conn-kind" className="text-sm text-slate-300">Kind</label>
            <select
              id="conn-kind"
              className="rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-sm"
              value={kind}
              onChange={(event) => setKind(event.target.value as ConnectionKind)}
            >
              <option value="fakevendor">fakevendor (demo)</option>
              <option value="openai">openai</option>
              <option value="anthropic">anthropic</option>
            </select>
          </div>
          <div className="flex flex-col gap-1.5 md:col-span-2">
            <label htmlFor="conn-secret" className="text-sm text-slate-300">Secret</label>
            <input
              id="conn-secret"
              type="password"
              className="rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-sm"
              value={secret}
              onChange={(event) => setSecret(event.target.value)}
              required
            />
          </div>
          <div className="md:col-span-2">
            <Button type="submit" disabled={createMutation.isPending}>
              {createMutation.isPending ? "Creating…" : "Create connection"}
            </Button>
          </div>
        </form>
      </section>

      <section className="rounded-2xl border border-slate-800 bg-slate-900/30 p-4 md:p-6">
        <h2 className="text-lg font-semibold text-white">Workspace connections</h2>

        {connectionsQuery.isLoading && (
          <p className="mt-4 text-slate-400" role="status">Loading connections…</p>
        )}

        {connectionsQuery.isError && (
          <p className="mt-4 text-rose-300" role="alert">
            {connectionsQuery.error instanceof ApiError
              ? connectionsQuery.error.userMessage()
              : "Could not load connections."}
          </p>
        )}

        {connectionsQuery.data && connectionsQuery.data.length === 0 && (
          <p className="mt-4 text-slate-400">No connections yet. Create one above.</p>
        )}

        {connectionsQuery.data && connectionsQuery.data.length > 0 && (
          <ul className="mt-4 divide-y divide-slate-800">
            {connectionsQuery.data.map((connection) => (
              <li key={connection.id} className="flex flex-col gap-3 py-4 md:flex-row md:items-center md:justify-between">
                <div>
                  <p className="font-medium text-white">{connection.name}</p>
                  <p className="mt-1 font-mono text-xs text-slate-500">{connection.id}</p>
                  <p className="mt-2 flex flex-wrap items-center gap-2 text-sm text-slate-400">
                    <span>{connection.kind}</span>
                    <ConnectionStatusBadge status={connection.status} />
                    {connection.credential?.fingerprint && (
                      <span className="font-mono text-xs text-slate-500">
                        fp {connection.credential.fingerprint.slice(0, 12)}…
                      </span>
                    )}
                  </p>
                  {connection.lastSuccessAt && (
                    <p className="mt-1 text-xs text-slate-500">
                      Last success {formatTimestamp(connection.lastSuccessAt, "medium")}
                    </p>
                  )}
                </div>
                <div className="flex flex-wrap gap-2">
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={() => verifyMutation.mutate(connection.id)}
                    disabled={verifyMutation.isPending}
                  >
                    Verify
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="text-slate-200"
                    onClick={() => backfillMutation.mutate(connection.id)}
                    disabled={backfillMutation.isPending}
                  >
                    Backfill 24h
                  </Button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
