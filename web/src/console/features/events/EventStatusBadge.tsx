"use client";

import { Badge } from "../../components/ui/Badge";
import type { EventStatus } from "../../domain/event";

const map: Record<EventStatus, "success" | "warning" | "danger"> = {
  processed: "success",
  pending: "warning",
  failed: "danger"
};

export function EventStatusBadge({ status }: { status: EventStatus }) {
  return <Badge variant={map[status]}>{status}</Badge>;
}
