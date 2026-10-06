package fakevendor

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/o-mid/contract-ops/api/internal/connectors"
)

func TestNormalizeMatchesGolden(t *testing.T) {
	payload, err := os.ReadFile("testdata/record.json")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := Connector{}.Normalize(connectors.RawRecord{ID: "fv_1", Payload: payload})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d", len(rows))
	}

	var want map[string]string
	golden, err := os.ReadFile("testdata/golden.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(golden, &want); err != nil {
		t.Fatal(err)
	}
	got := map[string]string{
		"providerName":      rows[0].ProviderName,
		"billingAccountId":  rows[0].BillingAccountID,
		"serviceName":       rows[0].ServiceName,
		"serviceCategory":   rows[0].ServiceCategory,
		"skuId":             rows[0].SKUID,
		"chargeCategory":    rows[0].ChargeCategory,
		"chargePeriodStart": rows[0].ChargePeriodStart.Format(time.RFC3339),
		"chargePeriodEnd":   rows[0].ChargePeriodEnd.Format(time.RFC3339),
		"billedCost":        rows[0].BilledCost,
		"effectiveCost":     rows[0].EffectiveCost,
		"billingCurrency":   rows[0].BillingCurrency,
		"usageQuantity":     rows[0].UsageQuantity,
		"usageUnit":         rows[0].UsageUnit,
		"sourceRecordId":    rows[0].SourceRecordID,
	}
	for key, value := range want {
		if got[key] != value {
			t.Errorf("%s = %q, want %q", key, got[key], value)
		}
	}
}

func TestDriftRejectsARenamedField(t *testing.T) {
	connector := New()
	page, err := connector.Fetch(context.Background(), connectors.Credential{Secret: `{"drift":true}`}, connectors.Window{}, connectors.Cursor{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := connector.Normalize(page.Records[0]); err == nil {
		t.Fatal("expected schema drift")
	}
}

func TestRateLimitDoesNotReturnRows(t *testing.T) {
	var slept time.Duration
	connector := Connector{sleep: func(d time.Duration) { slept = d }}
	_, err := connector.Fetch(context.Background(), connectors.Credential{Secret: `{"status":429,"retryAfter":2,"latency":"5ms"}`}, connectors.Window{}, connectors.Cursor{})
	var vendor *VendorError
	if !errorsAs(err, &vendor) || vendor.Status != 429 || vendor.RetryAfter != 2*time.Second {
		t.Fatalf("err = %#v", err)
	}
	if slept != 5*time.Millisecond {
		t.Fatalf("slept %s", slept)
	}
}

func errorsAs(err error, target **VendorError) bool {
	got, ok := err.(*VendorError)
	if ok {
		*target = got
	}
	return ok
}
