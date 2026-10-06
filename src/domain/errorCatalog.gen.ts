// Code generated from api/openapi/errors.yaml. DO NOT EDIT.

export type ErrorCatalogEntry = {
  code: string;
  severity: "error" | "warning";
  retryable: boolean;
  action: string;
  message: string;
};

export const errorCatalog = {
  auth_expired: { code: "auth_expired", severity: "error", retryable: false, action: "rotate_key_and_backfill", message: "Your {vendor} key expired on {date}. Data after that is missing." },
  auth_invalid: { code: "auth_invalid", severity: "error", retryable: false, action: "rotate_key", message: "{vendor} rejected the API key on {date}." },
  partial_sync: { code: "partial_sync", severity: "warning", retryable: true, action: "retry_failed_days", message: "{n} of {m} days synced." },
  permission_missing: { code: "permission_missing", severity: "error", retryable: false, action: "open_docs", message: "The key works but can't read billing data." },
  rate_limited: { code: "rate_limited", severity: "warning", retryable: true, action: "retry", message: "{vendor} is rate-limiting us. We'll retry at {time}." },
  schema_drift: { code: "schema_drift", severity: "error", retryable: false, action: "view_report", message: "{vendor} changed its data format. We paused writes to keep your numbers right." },
  vendor_unavailable: { code: "vendor_unavailable", severity: "warning", retryable: true, action: "retry", message: "{vendor} is down or unreachable since {time}." },
} as const satisfies Record<string, ErrorCatalogEntry>;

export type ErrorCode = keyof typeof errorCatalog;
