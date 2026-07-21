import type { EventFilters, EventPage, EventStatus, ServiceEvent } from "../domain/event";

const apiBaseUrl = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";

const eventStatuses: ReadonlySet<string> = new Set(["processed", "pending", "failed"]);

function buildQuery(filters: EventFilters): string {
  const parameters = new URLSearchParams();

  if (filters.query.trim()) {
    parameters.set("q", filters.query.trim());
  }

  if (filters.status !== "all") {
    parameters.set("status", filters.status);
  }

  return parameters.toString();
}

export function eventsStreamUrl(filters: EventFilters): string {
  const query = buildQuery(filters);
  return query
    ? `${apiBaseUrl}/v1/events/stream?${query}`
    : `${apiBaseUrl}/v1/events/stream`;
}

export function parseEventPage(value: unknown): EventPage {
  if (!isRecord(value)) {
    throw new Error("Invalid events response: expected an object");
  }

  if (typeof value.total !== "number" || !Number.isFinite(value.total) || value.total < 0) {
    throw new Error("Invalid events response: total must be a non-negative number");
  }

  if (!Array.isArray(value.events)) {
    throw new Error("Invalid events response: events must be an array");
  }

  const events = value.events.map((event, index) => parseServiceEvent(event, index));
  return { events, total: value.total };
}

function parseServiceEvent(value: unknown, index: number): ServiceEvent {
  if (!isRecord(value)) {
    throw new Error(`Invalid events response: event at index ${index} must be an object`);
  }

  const id = readString(value, "id", index);
  const source = readString(value, "source", index);
  const type = readString(value, "type", index);
  const occurredAt = readString(value, "occurredAt", index);
  const correlationId = readString(value, "correlationId", index);
  const statusValue = readString(value, "status", index);

  if (!eventStatuses.has(statusValue)) {
    throw new Error(`Invalid events response: event at index ${index} has unknown status`);
  }

  if (Number.isNaN(Date.parse(occurredAt))) {
    throw new Error(`Invalid events response: event at index ${index} has invalid occurredAt`);
  }

  return {
    id,
    source,
    type,
    status: statusValue as EventStatus,
    occurredAt,
    correlationId
  };
}

function readString(record: Record<string, unknown>, key: string, index: number): string {
  const value = record[key];
  if (typeof value !== "string" || value.trim() === "") {
    throw new Error(`Invalid events response: event at index ${index} is missing ${key}`);
  }
  return value;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

export async function getEvents(
  filters: EventFilters,
  signal?: AbortSignal
): Promise<EventPage> {
  const query = buildQuery(filters);
  const url = query ? `${apiBaseUrl}/v1/events?${query}` : `${apiBaseUrl}/v1/events`;

  const response = await fetch(url, {
    headers: { Accept: "application/json" },
    signal
  });

  if (!response.ok) {
    throw new Error(`Unable to load events (${response.status})`);
  }

  return parseEventPage(await response.json());
}
