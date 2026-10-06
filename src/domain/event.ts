// Types for the public event page. nextCursor is not part of this shape.
// The console renders the snapshot it was given and does not follow pages.
export type EventStatus = "processed" | "pending" | "failed";

export type ServiceEvent = {
  id: string;
  source: string;
  type: string;
  status: EventStatus;
  occurredAt: string;
  correlationId: string;
  connectionId?: string;
};

export type EventPage = {
  events: ServiceEvent[];
  total: number;
  nextCursor?: string;
};

export type EventFilters = {
  query: string;
  status: EventStatus | "all";
  connectionId: string;
};

export type EventsState =
  | { kind: "loading" }
  | { kind: "ready"; page: EventPage; lastUpdated: Date; source?: "stream" }
  | { kind: "error"; message: string };
