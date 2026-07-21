import { useDeferredValue, useId, useState } from "react";
import type { EventFilters, EventStatus } from "../domain/event";
import { EventTable } from "../features/events/EventTable";
import { EventToolbar } from "../features/events/EventToolbar";
import { useEvents } from "./useEvents";

const initialFilters: EventFilters = { query: "", status: "all" };

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
          Browse recent service events and inspect processing state.
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
