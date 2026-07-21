import type { ChangeEvent } from "react";
import type { EventFilters, EventStatus } from "../../domain/event";

type EventToolbarProps = {
  filters: EventFilters;
  onQueryChange: (query: string) => void;
  onStatusChange: (status: EventStatus | "all") => void;
  describedBy: string;
};

export function EventToolbar({
  filters,
  onQueryChange,
  onStatusChange,
  describedBy
}: EventToolbarProps) {
  function handleStatusChange(event: ChangeEvent<HTMLSelectElement>) {
    onStatusChange(event.target.value as EventStatus | "all");
  }

  return (
    <form className="event-toolbar" onSubmit={(event) => event.preventDefault()}>
      <div className="field">
        <label htmlFor="event-search">Find an event</label>
        <input
          id="event-search"
          type="search"
          value={filters.query}
          onChange={(event) => onQueryChange(event.target.value)}
          aria-describedby={describedBy}
          placeholder="Search source, type, or correlation ID"
        />
      </div>

      <div className="field">
        <label htmlFor="event-status">Processing status</label>
        <select
          id="event-status"
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
    </form>
  );
}
