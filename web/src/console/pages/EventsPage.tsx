"use client";

import { useId, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { listConnections } from "../api/connections";
import { useSettings } from "../app/useSettings";
import type { EventFilters } from "../domain/event";
import type { useEvents } from "../app/useEvents";
import { formatLastUpdated } from "../lib/format";
import { EventTable } from "../features/events/EventTable";
import { EventToolbar } from "../features/events/EventToolbar";

type EventsPageProps = {
  filters: EventFilters;
  onFiltersChange: (updater: (current: EventFilters) => EventFilters) => void;
  events: ReturnType<typeof useEvents>;
};

export function EventsPage({ filters, onFiltersChange, events }: EventsPageProps) {
  const statusMessageId = useId();
  const { settings, hasApiKey } = useSettings();
  const { state, retry, loadMore, loadingMore } = events;

  const connectionsQuery = useQuery({
    queryKey: ["connections", settings.apiBaseUrl],
    queryFn: ({ signal }) =>
      listConnections({
        apiBaseUrl: settings.apiBaseUrl,
        apiKey: settings.apiKey,
        signal
      }),
    enabled: hasApiKey,
    staleTime: 30_000
  });

  const statusCounts = useMemo(() => {
    if (state.kind !== "ready") {
      return null;
    }
    const counts = { processed: 0, pending: 0, failed: 0 };
    for (const event of state.page.events) {
      counts[event.status] += 1;
    }
    return counts;
  }, [state]);

  return (
    <div className="space-y-6">
      <header className="max-w-3xl">
        <h1 className="text-2xl font-semibold tracking-tight text-white md:text-3xl">
          Activity feed
        </h1>
        <p className="mt-2 text-slate-400">
          Live service events from the public stream. Filters sync to the URL so you can share a
          view with your team.
        </p>
      </header>

      <section
        aria-labelledby="events-heading"
        className="rounded-2xl border border-white/[0.06] bg-white/[0.02] p-4 md:p-6"
      >
        <div className="mb-6 flex flex-wrap items-end justify-between gap-4">
          <div>
            <h2 id="events-heading" className="text-lg font-semibold text-white">
              Recent events
            </h2>
            <p id={statusMessageId} className="mt-1 text-sm text-slate-400" aria-live="polite">
              {state.kind === "loading" && "Loading events"}
              {state.kind === "ready" &&
                `${state.page.total} event${state.page.total === 1 ? "" : "s"} match filters`}
              {state.kind === "error" && "Events could not be loaded"}
            </p>
            {state.kind === "ready" && (
              <p className="mt-1 text-xs text-slate-500">
                Last updated {formatLastUpdated(state.lastUpdated)} UTC
              </p>
            )}
          </div>
          {statusCounts && (
            <dl className="flex flex-wrap gap-3 text-xs text-slate-400">
              <div>
                <dt className="sr-only">Processed on page</dt>
                <dd>
                  <span className="text-emerald-400">{statusCounts.processed}</span> processed
                </dd>
              </div>
              <div>
                <dt className="sr-only">Pending on page</dt>
                <dd>
                  <span className="text-amber-400">{statusCounts.pending}</span> pending
                </dd>
              </div>
              <div>
                <dt className="sr-only">Failed on page</dt>
                <dd>
                  <span className="text-rose-400">{statusCounts.failed}</span> failed
                </dd>
              </div>
            </dl>
          )}
        </div>

        <EventToolbar
          filters={filters}
          onQueryChange={(query) => onFiltersChange((current) => ({ ...current, query }))}
          onStatusChange={(status) => onFiltersChange((current) => ({ ...current, status }))}
          onConnectionChange={(connectionId) =>
            onFiltersChange((current) => ({ ...current, connectionId }))
          }
          describedBy={statusMessageId}
          connections={connectionsQuery.data ?? []}
          showConnectionFilter={hasApiKey && (connectionsQuery.data?.length ?? 0) > 0}
        />

        <div className="mt-6">
          <EventTable
            state={state}
            onRetry={retry}
            onLoadMore={loadMore}
            loadingMore={loadingMore}
          />
        </div>
      </section>
    </div>
  );
}
