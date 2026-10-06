// Stream URL and the runtime check for one event page.
import type { EventFilters, EventPage, EventStatus, ServiceEvent } from "../domain/event";

const eventStatuses: ReadonlySet<string> = new Set(["processed", "pending", "failed"]);

export function buildEventsQuery(filters: EventFilters): string {
  const parameters = new URLSearchParams();

  if (filters.query.trim()) {
    parameters.set("q", filters.query.trim());
  }

  if (filters.status !== "all") {
    parameters.set("status", filters.status);
  }

  if (filters.connectionId.trim()) {
    parameters.set("connection_id", filters.connectionId.trim());
  }

  return parameters.toString();
}

export function eventsStreamUrl(apiBaseUrl: string, filters: EventFilters): string {
  const base = apiBaseUrl.replace(/\/$/, "");
  const query = buildEventsQuery(filters);
  return query ? `${base}/v1/events/stream?${query}` : `${base}/v1/events/stream`;
}

export function eventsPageUrl(apiBaseUrl: string, filters: EventFilters, cursor?: string): string {
  const base = apiBaseUrl.replace(/\/$/, "");
  const parameters = new URLSearchParams(buildEventsQuery(filters));
  if (cursor) {
    parameters.set("cursor", cursor);
  }
  const query = parameters.toString();
  return query ? `${base}/v1/events?${query}` : `${base}/v1/events`;
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
  const page: EventPage = { events, total: value.total };
  if (typeof value.nextCursor === "string" && value.nextCursor.trim()) {
    page.nextCursor = value.nextCursor;
  }
  return page;
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

  const event: ServiceEvent = {
    id,
    source,
    type,
    status: statusValue as EventStatus,
    occurredAt,
    correlationId
  };

  if (typeof value.connectionId === "string" && value.connectionId.trim()) {
    event.connectionId = value.connectionId;
  }

  return event;
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
  apiBaseUrl: string,
  filters: EventFilters,
  cursor?: string,
  signal?: AbortSignal
): Promise<EventPage> {
  const url = eventsPageUrl(apiBaseUrl, filters, cursor);

  const response = await fetch(url, {
    headers: { Accept: "application/json" },
    signal
  });

  if (!response.ok) {
    throw new Error(`Unable to load events (${response.status})`);
  }

  return parseEventPage(await response.json());
}
