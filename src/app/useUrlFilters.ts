import { useCallback, useEffect, useState } from "react";
import type { EventFilters, EventStatus } from "../domain/event";

export type AppView = "events" | "connections" | "settings";

function readView(): AppView {
  const value = new URLSearchParams(window.location.search).get("view");
  if (value === "connections" || value === "settings") {
    return value;
  }
  return "events";
}

function readEventFilters(): EventFilters {
  const params = new URLSearchParams(window.location.search);
  const status = params.get("status");
  const allowed: EventStatus[] = ["processed", "pending", "failed"];
  return {
    query: params.get("q") ?? "",
    status: allowed.includes(status as EventStatus) ? (status as EventStatus) : "all",
    connectionId: params.get("connection") ?? ""
  };
}

function writeUrl(view: AppView, filters: EventFilters) {
  const params = new URLSearchParams();
  if (view !== "events") {
    params.set("view", view);
  }
  if (filters.query.trim()) {
    params.set("q", filters.query.trim());
  }
  if (filters.status !== "all") {
    params.set("status", filters.status);
  }
  if (filters.connectionId.trim()) {
    params.set("connection", filters.connectionId.trim());
  }
  const next = params.toString();
  const path = next ? `?${next}` : window.location.pathname;
  window.history.replaceState(null, "", path);
}

export function useUrlNavigation() {
  const [view, setViewState] = useState<AppView>(() => readView());
  const [filters, setFiltersState] = useState<EventFilters>(() => readEventFilters());

  useEffect(() => {
    writeUrl(view, filters);
  }, [view, filters]);

  const setView = useCallback((next: AppView) => {
    setViewState(next);
  }, []);

  const setFilters = useCallback((updater: (current: EventFilters) => EventFilters) => {
    setFiltersState(updater);
  }, []);

  return { view, setView, filters, setFilters };
}
