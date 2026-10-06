import { Button } from "../../components/ui/Button";
import type { EventsState } from "../../domain/event";
import { formatTimestamp } from "../../lib/format";
import { EventStatusBadge } from "./EventStatusBadge";

type EventTableProps = {
  state: EventsState;
  onRetry: () => void;
  onLoadMore?: () => void;
  loadingMore?: boolean;
};

export function EventTable({ state, onRetry, onLoadMore, loadingMore }: EventTableProps) {
  if (state.kind === "loading") {
    return (
      <p className="rounded-xl border border-slate-800 bg-slate-900/50 p-6 text-slate-400" role="status">
        Loading service events…
      </p>
    );
  }

  if (state.kind === "error") {
    return (
      <div
        className="rounded-xl border border-rose-900/50 bg-rose-950/40 p-6 text-rose-100"
        role="alert"
      >
        <p className="font-medium">{state.message}</p>
        <Button variant="secondary" size="sm" className="mt-4" onClick={onRetry}>
          Try again
        </Button>
      </div>
    );
  }

  if (state.page.events.length === 0) {
    return (
      <div className="rounded-xl border border-slate-800 bg-slate-900/50 p-8 text-center">
        <h3 className="text-lg font-semibold text-white">No matching events</h3>
        <p className="mt-2 text-sm text-slate-400">
          Try a different search term, status, or connection filter.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="overflow-hidden rounded-xl border border-slate-800 bg-slate-900/40 shadow-xl shadow-black/20">
        <div className="overflow-x-auto">
          <table className="min-w-[48rem] w-full text-left text-sm">
            <thead className="border-b border-slate-800 bg-slate-900/80 text-xs uppercase tracking-wide text-slate-400">
              <tr>
                <th scope="col" className="px-4 py-3 font-semibold">Event</th>
                <th scope="col" className="px-4 py-3 font-semibold">Source</th>
                <th scope="col" className="px-4 py-3 font-semibold">Status</th>
                <th scope="col" className="px-4 py-3 font-semibold">Occurred (UTC)</th>
                <th scope="col" className="px-4 py-3 font-semibold">Correlation</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/80">
              {state.page.events.map((event) => (
                <tr key={event.id} className="hover:bg-slate-800/30">
                  <th scope="row" className="px-4 py-3 font-normal">
                    <span className="block font-medium text-white">{event.type}</span>
                    <span className="mt-0.5 block font-mono text-xs text-slate-500">{event.id}</span>
                    {event.connectionId && (
                      <span className="mt-1 block font-mono text-xs text-indigo-300/80">
                        {event.connectionId}
                      </span>
                    )}
                  </th>
                  <td className="px-4 py-3 text-slate-300">{event.source}</td>
                  <td className="px-4 py-3">
                    <EventStatusBadge status={event.status} />
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap text-slate-300">
                    {formatTimestamp(event.occurredAt, "medium")}
                  </td>
                  <td className="px-4 py-3">
                    <code className="rounded bg-slate-950 px-1.5 py-0.5 text-xs text-slate-300">
                      {event.correlationId}
                    </code>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {state.page.nextCursor && onLoadMore && (
        <div className="flex justify-center">
          <Button variant="secondary" onClick={onLoadMore} disabled={loadingMore}>
            {loadingMore ? "Loading…" : "Load older events"}
          </Button>
        </div>
      )}
    </div>
  );
}
