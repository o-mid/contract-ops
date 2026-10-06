"use client";

import { useQuery } from "@tanstack/react-query";
import { listCosts } from "../api/costs";
import { ApiError } from "../api/client";
import { useSettings } from "../app/useSettings";
import { formatTimestamp } from "../lib/format";

export function CostsPage() {
  const { settings, hasApiKey } = useSettings();
  const costsQuery = useQuery({
    queryKey: ["costs", settings.apiBaseUrl],
    queryFn: ({ signal }) =>
      listCosts({
        apiBaseUrl: settings.apiBaseUrl,
        apiKey: settings.apiKey,
        limit: 50,
        signal
      }),
    enabled: hasApiKey
  });

  if (!hasApiKey) {
    return (
      <div className="max-w-xl space-y-4">
        <h1 className="text-2xl font-semibold text-white">Costs</h1>
        <p className="text-slate-400">
          Add your workspace API key in Settings to list normalized billing rows from completed sync
          jobs.
        </p>
      </div>
    );
  }

  const rows = costsQuery.data?.costs ?? [];
  const total = costsQuery.data?.total ?? 0;
  return (
    <div className="space-y-8">
      <header className="max-w-3xl">
        <h1 className="text-2xl font-semibold tracking-tight text-white md:text-3xl">Costs</h1>
        <p className="mt-2 text-slate-400">
          FOCUS-shaped billing rows persisted when sync jobs commit a page. Filter by connection from
          the activity feed or queue a backfill to ingest demo data.
        </p>
      </header>

      <section className="rounded-2xl border border-slate-800 bg-slate-900/30 p-4 md:p-6">
        <div className="flex flex-wrap items-baseline justify-between gap-2">
          <h2 className="text-lg font-semibold text-white">Billing rows</h2>
          {costsQuery.data && (
            <p className="text-sm text-slate-500">
              {total} row{total === 1 ? "" : "s"} in workspace
            </p>
          )}
        </div>

        {costsQuery.isLoading && (
          <p className="mt-4 text-slate-400" role="status">Loading costs…</p>
        )}

        {costsQuery.isError && (
          <p className="mt-4 text-rose-300" role="alert">
            {costsQuery.error instanceof ApiError
              ? costsQuery.error.userMessage()
              : "Could not load costs."}
          </p>
        )}

        {costsQuery.data && rows.length === 0 && (
          <p className="mt-4 text-slate-400">
            No cost rows yet. Verify a connection and run a backfill, or wait for the worker to
            complete a scheduled sync.
          </p>
        )}

        {rows.length > 0 && (
          <div className="mt-4 overflow-x-auto">
            <table className="w-full min-w-[640px] text-left text-sm">
              <thead>
                <tr className="border-b border-slate-800 text-slate-500">
                  <th className="py-2 pr-4 font-medium">Period</th>
                  <th className="py-2 pr-4 font-medium">Provider</th>
                  <th className="py-2 pr-4 font-medium">Service</th>
                  <th className="py-2 pr-4 font-medium text-right">Billed</th>
                  <th className="py-2 font-medium">Connection</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800/80">
                {rows.map((row) => (
                  <tr key={row.id}>
                    <td className="py-3 pr-4 text-slate-300">
                      {formatTimestamp(row.chargePeriodStart, "medium")}
                    </td>
                    <td className="py-3 pr-4 text-slate-300">{row.providerName}</td>
                    <td className="py-3 pr-4 text-white">{row.serviceName}</td>
                    <td className="py-3 pr-4 text-right font-mono text-indigo-100">
                      {row.billingCurrency} {row.billedCost}
                    </td>
                    <td className="py-3 font-mono text-xs text-slate-500">
                      {row.connectionId.slice(0, 18)}…
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

      </section>
    </div>
  );
}
