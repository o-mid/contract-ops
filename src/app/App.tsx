// Filter state lives here. The deferred copy is what opens the stream,
// so a keystroke does not reconnect on every character.
import { useDeferredValue, useId, useState } from "react";
import type { EventFilters, EventStatus } from "../domain/event";
import { EventTable } from "../features/events/EventTable";
import { EventToolbar } from "../features/events/EventToolbar";
import { useEvents } from "./useEvents";

const initialFilters: EventFilters = { query: "", status: "all" };

function formatLastUpdated(value: Date) {
  return new Intl.DateTimeFormat("en-GB", {
    dateStyle: "medium",
    timeStyle: "medium",
    timeZone: "UTC"
  }).format(value);
}

export function App() {
  const [filters, setFilters] = useState(initialFilters);
  const deferredFilters = useDeferredValue(filters);
  const statusMessageId = useId();
  const { state, retry } = useEvents(deferredFilters);

  function setQuery(query: string) {
    setFilters((current) => ({ ...current, query }));
  }

  function setStatus(status: EventStatus | "all") {
    setFilters((current) => ({ ...current, status }));
  }

  return (
    <main className="app-shell">
      <header className="page-header">
        <p className="eyebrow">Contract Ops</p>
        <h1>Event console</h1>
        <p className="page-summary">
          Browse recent service events and inspect processing state. New demo events
          appear live over a server-sent stream until eight events are available.
        </p>
      </header>

      <section aria-labelledby="events-heading" className="event-panel">
        <div className="panel-heading">
          <div>
            <h2 id="events-heading">Recent events</h2>
            <p id={statusMessageId} className="result-summary" aria-live="polite">
              {state.kind === "loading" && "Loading events"}
              {state.kind === "ready" &&
                `${state.page.total} event${state.page.total === 1 ? "" : "s"} found`}
              {state.kind === "error" && "Events could not be loaded"}
            </p>
            {state.kind === "ready" && (
              <p className="last-updated">
                Last updated {formatLastUpdated(state.lastUpdated)} UTC
              </p>
            )}
          </div>
        </div>

        <EventToolbar
          filters={filters}
          onQueryChange={setQuery}
          onStatusChange={setStatus}
          describedBy={statusMessageId}
        />

        <EventTable state={state} onRetry={retry} />
      </section>
    </main>
  );
}
