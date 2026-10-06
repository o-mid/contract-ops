import type { ChangeEvent } from "react";
import type { Connection } from "../../domain/connection";
import type { EventFilters, EventStatus } from "../../domain/event";

type EventToolbarProps = {
  filters: EventFilters;
  onQueryChange: (query: string) => void;
  onStatusChange: (status: EventStatus | "all") => void;
  onConnectionChange: (connectionId: string) => void;
  describedBy: string;
  connections: Connection[];
  showConnectionFilter: boolean;
};

const fieldClass =
  "w-full min-h-[2.75rem] rounded-lg border border-slate-700 bg-slate-900/80 px-3 py-2 text-sm text-slate-100 placeholder:text-slate-500 focus:border-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/30";

export function EventToolbar({
  filters,
  onQueryChange,
  onStatusChange,
  onConnectionChange,
  describedBy,
  connections,
  showConnectionFilter
}: EventToolbarProps) {
  function handleStatusChange(event: ChangeEvent<HTMLSelectElement>) {
    onStatusChange(event.target.value as EventStatus | "all");
  }

  function handleConnectionChange(event: ChangeEvent<HTMLSelectElement>) {
    onConnectionChange(event.target.value);
  }

  return (
    <form
      className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3"
      onSubmit={(event) => event.preventDefault()}
    >
      <div className="flex flex-col gap-1.5 sm:col-span-2 lg:col-span-1">
        <label htmlFor="event-search" className="text-sm font-medium text-slate-300">
          Search
        </label>
        <input
          id="event-search"
          type="search"
          className={fieldClass}
          value={filters.query}
          onChange={(event) => onQueryChange(event.target.value)}
          aria-describedby={describedBy}
          placeholder="Source, type, correlation ID…"
        />
      </div>

      <div className="flex flex-col gap-1.5">
        <label htmlFor="event-status" className="text-sm font-medium text-slate-300">
          Status
        </label>
        <select
          id="event-status"
          className={fieldClass}
          value={filters.status}
          onChange={handleStatusChange}
          aria-describedby={describedBy}
        >
          <option value="all">All statuses</option>
          <option value="processed">Processed</option>
          <option value="pending">Pending</option>
          <option value="failed">Failed</option>
        </select>
      </div>

      {showConnectionFilter && (
        <div className="flex flex-col gap-1.5">
          <label htmlFor="event-connection" className="text-sm font-medium text-slate-300">
            Connection
          </label>
          <select
            id="event-connection"
            className={fieldClass}
            value={filters.connectionId}
            onChange={handleConnectionChange}
            aria-describedby={describedBy}
          >
            <option value="">All connections</option>
            {connections.map((connection) => (
              <option key={connection.id} value={connection.id}>
                {connection.name} ({connection.kind})
              </option>
            ))}
          </select>
        </div>
      )}
    </form>
  );
}
