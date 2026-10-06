// One EventSource per filter change. EventSource cannot set Authorization,
// so this hook only calls the public feed.
import { useCallback, useEffect, useState } from "react";
import { eventsStreamUrl, getEvents, parseEventPage } from "../api/events";
import type { EventFilters, EventsState } from "../domain/event";

export type { EventsState };

export function useEvents(apiBaseUrl: string, filters: EventFilters) {
  const [state, setState] = useState<EventsState>({ kind: "loading" });
  const [retryToken, setRetryToken] = useState(0);
  const [loadingMore, setLoadingMore] = useState(false);
  const { query, status, connectionId } = filters;

  useEffect(() => {
    let active = true;
    setState({ kind: "loading" });

    const source = new EventSource(eventsStreamUrl(apiBaseUrl, filters));

    source.onmessage = (message) => {
      if (!active) {
        return;
      }

      try {
        const page = parseEventPage(JSON.parse(message.data) as unknown);
        setState({ kind: "ready", page, lastUpdated: new Date(), source: "stream" });
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
    // filters fields are listed explicitly so deferred query updates do not reconnect early.
    // eslint-disable-next-line react-hooks/exhaustive-deps -- connectionId, query, status
  }, [apiBaseUrl, query, status, connectionId, retryToken]);

  const loadMore = useCallback(async () => {
    if (state.kind !== "ready" || !state.page.nextCursor || loadingMore) {
      return;
    }

    setLoadingMore(true);
    try {
      const next = await getEvents(apiBaseUrl, filters, state.page.nextCursor);
      setState((current) => {
        if (current.kind !== "ready") {
          return current;
        }
        const seen = new Set(current.page.events.map((event) => event.id));
        const merged = [...current.page.events];
        for (const event of next.events) {
          if (!seen.has(event.id)) {
            merged.push(event);
            seen.add(event.id);
          }
        }
        return {
          kind: "ready",
          page: {
            events: merged,
            total: current.page.total,
            nextCursor: next.nextCursor
          },
          lastUpdated: current.lastUpdated,
          source: "stream"
        };
      });
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : "Unable to load more events";
      setState({ kind: "error", message });
    } finally {
      setLoadingMore(false);
    }
  }, [apiBaseUrl, filters, loadingMore, state]);

  return {
    state,
    retry: () => setRetryToken((token) => token + 1),
    loadMore,
    loadingMore
  };
}
