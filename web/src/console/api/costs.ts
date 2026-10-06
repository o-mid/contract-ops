import { apiRequest } from "./client";
import type { CostPage, CostRow } from "../domain/cost";

function parseCostRow(value: unknown, index: number): CostRow {
  if (!isRecord(value)) {
    throw new Error(`Invalid cost at index ${index}`);
  }
  return {
    id: readString(value, "id", index),
    connectionId: readString(value, "connectionId", index),
    jobId: optionalString(value, "jobId"),
    batchId: optionalString(value, "batchId"),
    providerName: readString(value, "providerName", index),
    billingAccountId: optionalString(value, "billingAccountId"),
    serviceName: readString(value, "serviceName", index),
    serviceCategory: optionalString(value, "serviceCategory"),
    skuId: optionalString(value, "skuId"),
    chargeCategory: optionalString(value, "chargeCategory"),
    chargePeriodStart: readString(value, "chargePeriodStart", index),
    chargePeriodEnd: readString(value, "chargePeriodEnd", index),
    billedCost: readString(value, "billedCost", index),
    effectiveCost: readString(value, "effectiveCost", index),
    billingCurrency: readString(value, "billingCurrency", index),
    usageQuantity: optionalString(value, "usageQuantity"),
    usageUnit: optionalString(value, "usageUnit"),
    sourceRecordId: optionalString(value, "sourceRecordId"),
    ingestedAt: optionalString(value, "ingestedAt")
  };
}

export async function listCosts(
  options: {
    apiBaseUrl: string;
    apiKey: string;
    connectionId?: string;
    cursor?: string;
    limit?: number;
    signal?: AbortSignal;
  }
): Promise<CostPage> {
  const params = new URLSearchParams();
  if (options.connectionId?.trim()) {
    params.set("connection_id", options.connectionId.trim());
  }
  if (options.cursor) {
    params.set("cursor", options.cursor);
  }
  if (options.limit) {
    params.set("limit", String(options.limit));
  }
  const query = params.toString();
  const path = query ? `/v1/costs?${query}` : "/v1/costs";
  const payload = await apiRequest<{
    costs: unknown[];
    total: number;
    nextCursor?: string;
  }>(path, options);
  if (!Array.isArray(payload.costs)) {
    throw new Error("Invalid costs response");
  }
  return {
    costs: payload.costs.map((row, index) => parseCostRow(row, index)),
    total: payload.total,
    nextCursor: payload.nextCursor
  };
}

function readString(record: Record<string, unknown>, key: string, index: number): string {
  const value = record[key];
  if (typeof value !== "string" || value.trim() === "") {
    throw new Error(`Missing ${key} at index ${index}`);
  }
  return value;
}

function optionalString(record: Record<string, unknown>, key: string): string | undefined {
  const value = record[key];
  return typeof value === "string" && value ? value : undefined;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}
