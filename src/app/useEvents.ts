// One EventSource per filter change. EventSource cannot set Authorization,
// so this hook only calls the public feed.
import { useEffect, useState } from "react";
import { eventsStreamUrl, parseEventPage } from "../api/events";
import type { EventFilters, EventsState } from "../domain/event";

export type { EventsState };

export function useEvents(filters: EventFilters) {
  const [state, setState] = useState<EventsState>({ kind: "loading" });
  const [retryToken, setRetryToken] = useState(0);
  const { query, status } = filters;

  useEffect(() => {
    let active = true;
    setState({ kind: "loading" });

    const source = new EventSource(eventsStreamUrl({ query, status }));

    source.onmessage = (message) => {
      if (!active) {
        return;
      }

      try {
        const page = parseEventPage(JSON.parse(message.data) as unknown);
        setState({ kind: "ready", page, lastUpdated: new Date() });
      } catch (error: unknown) {
        const detail =
          error instanceof Error ? error.message : "Invalid events stream payload";
        active = false;
        setState({ kind: "error", message: detail });
        source.close();
      }
    };

    source.onerror = () => {
      if (!active) {
        return;
      }
      active = false;
      setState({
        kind: "error",
        message: "Unable to connect to the events stream"
      });
      source.close();
    };

    return () => {
      active = false;
      source.close();
    };
  }, [query, status, retryToken]);

  return {
    state,
    retry: () => setRetryToken((token) => token + 1)
  };
}
