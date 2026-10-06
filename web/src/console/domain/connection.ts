export type ConnectionStatus =
  | "healthy"
  | "degraded"
  | "failing"
  | "needs_auth"
  | "paused";

export type ConnectionKind = "fakevendor" | "openai" | "anthropic";

export type Credential = {
  fingerprint: string;
  expiresAt?: string;
};

export type Connection = {
  id: string;
  kind: ConnectionKind;
  name: string;
  status: ConnectionStatus;
  statusReasonCode?: string;
  lastSuccessAt?: string;
  lastErrorAt?: string;
  credential?: Credential;
};

export type SyncJob = {
  id: string;
  connectionId: string;
  kind: string;
  windowStart: string;
  windowEnd: string;
  status: string;
};
