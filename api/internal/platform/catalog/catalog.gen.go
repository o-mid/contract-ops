// Code generated from openapi/errors.yaml. DO NOT EDIT.

package catalog

type Entry struct {
	Code      string
	Severity  string
	Retryable bool
	Action    string
	Message   string
}

var ByCode = map[string]Entry{
	"auth_expired":       {Code: "auth_expired", Severity: "error", Retryable: false, Action: "rotate_key_and_backfill", Message: "Your {vendor} key expired on {date}. Data after that is missing."},
	"auth_invalid":       {Code: "auth_invalid", Severity: "error", Retryable: false, Action: "rotate_key", Message: "{vendor} rejected the API key on {date}."},
	"partial_sync":       {Code: "partial_sync", Severity: "warning", Retryable: true, Action: "retry_failed_days", Message: "{n} of {m} days synced."},
	"permission_missing": {Code: "permission_missing", Severity: "error", Retryable: false, Action: "open_docs", Message: "The key works but can't read billing data."},
	"rate_limited":       {Code: "rate_limited", Severity: "warning", Retryable: true, Action: "retry", Message: "{vendor} is rate-limiting us. We'll retry at {time}."},
	"schema_drift":       {Code: "schema_drift", Severity: "error", Retryable: false, Action: "view_report", Message: "{vendor} changed its data format. We paused writes to keep your numbers right."},
	"vendor_unavailable": {Code: "vendor_unavailable", Severity: "warning", Retryable: true, Action: "retry", Message: "{vendor} is down or unreachable since {time}."},
}
