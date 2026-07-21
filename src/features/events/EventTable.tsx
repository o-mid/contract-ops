import type { ServiceEvent } from "../../domain/event";

type EventsState =
  | { kind: "loading" }
  | { kind: "ready"; page: { events: ServiceEvent[]; total: number } }
  | { kind: "error"; message: string };

type EventTableProps = {
  state: EventsState;
  onRetry: () => void;
};

function formatTimestamp(value: string) {
  return new Intl.DateTimeFormat("en-GB", {
    dateStyle: "medium",
    timeStyle: "short",
    timeZone: "UTC"
  }).format(new Date(value));
}

export function EventTable({ state, onRetry }: EventTableProps) {
  if (state.kind === "loading") {
    return <p className="notice" role="status">Loading service events…</p>;
  }

  if (state.kind === "error") {
    return (
      <div className="notice notice-error" role="alert">
        <p>{state.message}</p>
        <button type="button" onClick={onRetry}>Try again</button>
      </div>
    );
  }

  if (state.page.events.length === 0) {
    return (
      <div className="notice">
        <h3>No matching events</h3>
        <p>Try a different search term or processing status.</p>
      </div>
    );
  }

  return (
    <div className="table-scroll">
      <table>
        <thead>
          <tr>
            <th scope="col">Event</th>
            <th scope="col">Source</th>
            <th scope="col">Status</th>
            <th scope="col">Occurred</th>
            <th scope="col">Correlation ID</th>
          </tr>
        </thead>
        <tbody>
          {state.page.events.map((event) => (
            <tr key={event.id}>
              <th scope="row">
                <span className="event-type">{event.type}</span>
                <span className="event-id">{event.id}</span>
              </th>
              <td>{event.source}</td>
              <td>
                <span className={`status status-${event.status}`}>{event.status}</span>
              </td>
              <td>{formatTimestamp(event.occurredAt)}</td>
              <td><code>{event.correlationId}</code></td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
