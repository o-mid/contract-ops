import type { EventFilters, EventPage } from "../domain/event";

const apiBaseUrl = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";

export async function getEvents(
  filters: EventFilters,
  signal?: AbortSignal
): Promise<EventPage> {
  const parameters = new URLSearchParams();

  if (filters.query.trim()) {
    parameters.set("q", filters.query.trim());
  }

  if (filters.status !== "all") {
    parameters.set("status", filters.status);
  }

  const response = await fetch(`${apiBaseUrl}/v1/events?${parameters.toString()}`, {
    headers: { Accept: "application/json" },
    signal
  });

  if (!response.ok) {
    throw new Error(`Unable to load events (${response.status})`);
  }

  return (await response.json()) as EventPage;
}
