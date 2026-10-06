// Package fakevendor is the in-process connector used by demos and tests.
// Its secret may be any non-empty string, or a JSON fault document.
package fakevendor

import (
	"context"
	"embed"
	"encoding/json"
	"strings"
	"time"

	"github.com/o-mid/contract-ops/api/internal/connectors"
)

//go:embed testdata/record.json
var fixture embed.FS

const kind = "fakevendor"

type Connector struct {
	sleep func(time.Duration)
}

func New() Connector {
	return Connector{sleep: time.Sleep}
}

func (c Connector) Kind() string { return kind }

func (Connector) Describe() connectors.Descriptor {
	return connectors.Descriptor{
		Kind:        kind,
		DisplayName: "Fake vendor",
		Fields:      []connectors.Field{{Name: "secret", Required: true, Secret: true}},
	}
}

func (Connector) Verify(_ context.Context, cred connectors.Credential) error {
	if strings.TrimSpace(cred.Secret) == "" {
		return connectors.ErrRejected
	}
	if strings.HasPrefix(strings.TrimSpace(cred.Secret), "{") {
		var faults faults
		if err := json.Unmarshal([]byte(cred.Secret), &faults); err != nil {
			return connectors.ErrRejected
		}
	}
	return nil
}

func (c Connector) Fetch(_ context.Context, cred connectors.Credential, _ connectors.Window, _ connectors.Cursor) (connectors.Page, error) {
	faults, err := parseFaults(cred.Secret)
	if err != nil {
		return connectors.Page{}, err
	}
	if wait := faults.wait(); wait > 0 && c.sleep != nil {
		c.sleep(wait)
	}
	if faults.Status == 429 || faults.Status >= 500 {
		retry := time.Second
		if faults.RetryAfter > 0 {
			retry = time.Duration(faults.RetryAfter) * time.Second
		}
		return connectors.Page{}, &VendorError{Status: faults.Status, RetryAfter: retry}
	}

	payload, err := fixture.ReadFile("testdata/record.json")
	if err != nil {
		return connectors.Page{}, err
	}
	if faults.Drift {
		payload = []byte(strings.Replace(string(payload), `"service"`, `"service_v2"`, 1))
	}
	page := connectors.Page{
		Records: []connectors.RawRecord{{ID: "fv_1", Payload: payload}},
		Done:    true,
	}
	if faults.Restate {
		page.Records = append(page.Records, connectors.RawRecord{ID: "fv_1", Payload: payload})
	}
	return page, nil
}

func (Connector) Normalize(raw connectors.RawRecord) ([]connectors.CostRow, error) {
	var body record
	if err := json.Unmarshal(raw.Payload, &body); err != nil {
		return nil, err
	}
	if body.Service == "" {
		return nil, &DriftError{Path: "service"}
	}
	start, err := time.Parse(time.RFC3339, body.Start)
	if err != nil {
		return nil, err
	}
	end, err := time.Parse(time.RFC3339, body.End)
	if err != nil {
		return nil, err
	}
	return []connectors.CostRow{{
		ProviderName:      kind,
		BillingAccountID:  "fake-account",
		ServiceName:       body.Service,
		ServiceCategory:   "AI and Machine Learning",
		SKUID:             body.SKU,
		ChargeCategory:    "Usage",
		ChargePeriodStart: start,
		ChargePeriodEnd:   end,
		BilledCost:        body.Billed,
		EffectiveCost:     body.Billed,
		BillingCurrency:   body.Currency,
		UsageQuantity:     body.Quantity,
		UsageUnit:         body.Unit,
		SourceRecordID:    body.ID,
	}}, nil
}

type faults struct {
	Latency    string `json:"latency"`
	Status     int    `json:"status"`
	RetryAfter int    `json:"retryAfter"`
	Drift      bool   `json:"drift"`
	Restate    bool   `json:"restate"`
}

func (f faults) wait() time.Duration {
	if f.Latency == "" {
		return 0
	}
	parsed, err := time.ParseDuration(f.Latency)
	if err != nil {
		return 0
	}
	return parsed
}

type record struct {
	ID       string `json:"id"`
	Service  string `json:"service"`
	SKU      string `json:"sku"`
	Start    string `json:"start"`
	End      string `json:"end"`
	Billed   string `json:"billed"`
	Currency string `json:"currency"`
	Quantity string `json:"quantity"`
	Unit     string `json:"unit"`
}

type VendorError struct {
	Status     int
	RetryAfter time.Duration
}

func (e *VendorError) Error() string {
	return "vendor request failed"
}

type DriftError struct {
	Path string
}

func (e *DriftError) Error() string {
	return "schema drift at " + e.Path
}

func (e *DriftError) SchemaDrift() string { return e.Path }

func (e *VendorError) Code() string {
	if e.Status == 429 {
		return "rate_limited"
	}
	return "vendor_unavailable"
}

func (e *VendorError) GetRetryAfter() time.Duration { return e.RetryAfter }

func parseFaults(secret string) (faults, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return faults{}, connectors.ErrRejected
	}
	if !strings.HasPrefix(secret, "{") {
		return faults{}, nil
	}
	var parsed faults
	if err := json.Unmarshal([]byte(secret), &parsed); err != nil {
		return faults{}, connectors.ErrRejected
	}
	return parsed, nil
}
