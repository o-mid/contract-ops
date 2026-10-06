package connections

import "testing"

func TestApplyHealthGraph(t *testing.T) {
	tests := []struct {
		name    string
		current Status
		code    string
		want    Status
	}{
		{name: "success", current: StatusDegraded, code: "", want: StatusHealthy},
		{name: "auth from healthy", current: StatusHealthy, code: "auth_invalid", want: StatusNeedsAuth},
		{name: "expired", current: StatusHealthy, code: "auth_expired", want: StatusNeedsAuth},
		{name: "permission", current: StatusFailing, code: "permission_missing", want: StatusNeedsAuth},
		{name: "drift from healthy", current: StatusHealthy, code: "schema_drift", want: StatusDegraded},
		{name: "rate limit once", current: StatusHealthy, code: "rate_limited", want: StatusDegraded},
		{name: "rate limit again", current: StatusDegraded, code: "vendor_unavailable", want: StatusFailing},
		{name: "outage while unauthorized", current: StatusNeedsAuth, code: "vendor_unavailable", want: StatusNeedsAuth},
		{name: "paused ignores jobs", current: StatusPaused, code: "", want: StatusPaused},
		{name: "paused ignores drift", current: StatusPaused, code: "schema_drift", want: StatusPaused},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := Apply(test.current, test.code); got != test.want {
				t.Fatalf("Apply(%s, %q) = %s, want %s", test.current, test.code, got, test.want)
			}
		})
	}
}
