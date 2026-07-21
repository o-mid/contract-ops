import { useEffect, useState } from "react";
import { getEvents } from "../api/events";
import type { EventFilters, EventPage } from "../domain/event";

type EventsState =
  | { kind: "loading" }
  | { kind: "ready"; page: EventPage }
  | { kind: "error"; message: string };

export function useEvents(filters: EventFilters) {
  const [state, setState] = useState<EventsState>({ kind: "loading" });
  const [retryToken, setRetryToken] = useState(0);
  const { query, status } = filters;

  useEffect(() => {
    const controller = new AbortController();
    setState({ kind: "loading" });

    getEvents({ query, status }, controller.signal)
      .then((page) => {
        if (!controller.signal.aborted) {
          setState({ kind: "ready", page });
        }
      })
      .catch((error: unknown) => {
        if (!controller.signal.aborted) {
          const message =
            error instanceof Error ? error.message : "Unable to load service events";
          setState({ kind: "error", message });
        }
      });

    return () => controller.abort();
  }, [query, status, retryToken]);

  return {
    state,
    retry: () => setRetryToken((token) => token + 1)
  };
}
