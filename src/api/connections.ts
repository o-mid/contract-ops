import { apiRequest } from "./client";
import type { Connection, ConnectionKind, SyncJob } from "../domain/connection";

function parseConnection(value: unknown, index: number): Connection {
  if (!isRecord(value)) {
    throw new Error(`Invalid connection at index ${index}`);
  }
  const id = readString(value, "id", index);
  const kind = readString(value, "kind", index) as ConnectionKind;
  const name = readString(value, "name", index);
  const status = readString(value, "status", index) as Connection["status"];
  const connection: Connection = { id, kind, name, status };
  if (typeof value.statusReasonCode === "string" && value.statusReasonCode) {
    connection.statusReasonCode = value.statusReasonCode;
  }
  if (typeof value.lastSuccessAt === "string") {
    connection.lastSuccessAt = value.lastSuccessAt;
  }
  if (typeof value.lastErrorAt === "string") {
    connection.lastErrorAt = value.lastErrorAt;
  }
  if (isRecord(value.credential) && typeof value.credential.fingerprint === "string") {
    connection.credential = {
      fingerprint: value.credential.fingerprint,
      expiresAt:
        typeof value.credential.expiresAt === "string" ? value.credential.expiresAt : undefined
    };
  }
  return connection;
}

export async function listConnections(options: {
  apiBaseUrl: string;
  apiKey: string;
  signal?: AbortSignal;
}): Promise<Connection[]> {
  const payload = await apiRequest<{ connections: unknown[] }>("/v1/connections", options);
  if (!Array.isArray(payload.connections)) {
    throw new Error("Invalid connections response");
  }
  return payload.connections.map((item, index) => parseConnection(item, index));
}

export async function createConnection(
  input: { kind: ConnectionKind; name: string; secret: string },
  options: { apiBaseUrl: string; apiKey: string }
): Promise<Connection> {
  const payload = await apiRequest<unknown>("/v1/connections", {
    ...options,
    method: "POST",
    body: input
  });
  return parseConnection(payload, 0);
}

export async function verifyConnection(
  id: string,
  options: { apiBaseUrl: string; apiKey: string }
): Promise<Connection> {
  const payload = await apiRequest<unknown>(`/v1/connections/${encodeURIComponent(id)}/verify`, {
    ...options,
    method: "POST"
  });
  return parseConnection(payload, 0);
}

export async function backfillConnection(
  id: string,
  start: string,
  end: string,
  options: { apiBaseUrl: string; apiKey: string }
): Promise<SyncJob[]> {
  const payload = await apiRequest<{ jobs: unknown[] }>(
    `/v1/connections/${encodeURIComponent(id)}/backfill`,
    {
      ...options,
      method: "POST",
      body: { start, end }
    }
  );
  if (!Array.isArray(payload.jobs)) {
    throw new Error("Invalid backfill response");
  }
  return payload.jobs.map((job, index) => parseSyncJob(job, index));
}

function parseSyncJob(value: unknown, index: number): SyncJob {
  if (!isRecord(value)) {
    throw new Error(`Invalid job at index ${index}`);
  }
  return {
    id: readString(value, "id", index),
    connectionId: readString(value, "connectionId", index),
    kind: readString(value, "kind", index),
    windowStart: readString(value, "windowStart", index),
    windowEnd: readString(value, "windowEnd", index),
    status: readString(value, "status", index)
  };
}

function readString(record: Record<string, unknown>, key: string, index: number): string {
  const value = record[key];
  if (typeof value !== "string" || value.trim() === "") {
    throw new Error(`Missing ${key} at index ${index}`);
  }
  return value;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}
