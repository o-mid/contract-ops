import { Badge } from "../../components/ui/Badge";
import type { ConnectionStatus } from "../../domain/connection";

const map: Record<
  ConnectionStatus,
  "success" | "warning" | "danger" | "muted" | "default"
> = {
  healthy: "success",
  degraded: "warning",
  failing: "danger",
  needs_auth: "warning",
  paused: "muted"
};

export function ConnectionStatusBadge({ status }: { status: ConnectionStatus }) {
  return <Badge variant={map[status]}>{status.replace("_", " ")}</Badge>;
}
