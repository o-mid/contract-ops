package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/o-mid/contract-ops/api/internal/connectors"
	"github.com/o-mid/contract-ops/api/internal/connectors/httpclient"
)

const kind = "openai"

type Connector struct {
	Base   string
	Client httpclient.Client
}

func New() Connector {
	return Connector{Base: "https://api.openai.com"}
}

func (c Connector) Kind() string { return kind }

func (Connector) Describe() connectors.Descriptor {
	return connectors.Descriptor{
		Kind:        kind,
		DisplayName: "OpenAI",
		DocsURL:     "https://platform.openai.com/docs/api-reference/usage/costs",
		Fields:      []connectors.Field{{Name: "api_key", Required: true, Secret: true}},
	}
}

func (c Connector) Verify(ctx context.Context, cred connectors.Credential) error {
	response, err := c.do(ctx, cred, http.MethodGet, "/v1/models", nil)
	if err != nil {
		return err
	}
	defer httpclient.Discard(response.Body)
	if response.StatusCode == http.StatusUnauthorized {
		return connectors.ErrRejected
	}
	if response.StatusCode >= 400 {
		return fmt.Errorf("openai verify returned %d", response.StatusCode)
	}
	return nil
}

func (c Connector) Fetch(ctx context.Context, cred connectors.Credential, window connectors.Window, _ connectors.Cursor) (connectors.Page, error) {
	path := fmt.Sprintf(
		"/v1/organization/costs?start_time=%d&end_time=%d&limit=100",
		window.Start.Unix(),
		window.End.Unix(),
	)
	response, err := c.do(ctx, cred, http.MethodGet, path, nil)
	if err != nil {
		return connectors.Page{}, err
	}
	body, err := io.ReadAll(response.Body)
	httpclient.Discard(response.Body)
	if response.StatusCode == http.StatusUnauthorized {
		return connectors.Page{}, connectors.ErrRejected
	}
	if response.StatusCode == http.StatusTooManyRequests {
		return connectors.Page{}, &vendorError{status: response.StatusCode, retryAfter: response.Header.Get("Retry-After")}
	}
	if response.StatusCode >= 500 {
		return connectors.Page{}, &vendorError{status: response.StatusCode, retryAfter: response.Header.Get("Retry-After")}
	}
	if response.StatusCode >= 400 {
		return connectors.Page{}, fmt.Errorf("openai costs returned %d", response.StatusCode)
	}
	if err != nil {
		return connectors.Page{}, err
	}
	return connectors.Page{
		Records: []connectors.RawRecord{{ID: "openai_costs", Payload: body}},
		Done:    true,
	}, nil
}

func (Connector) Normalize(raw connectors.RawRecord) ([]connectors.CostRow, error) {
	var payload costsResponse
	if err := json.Unmarshal(raw.Payload, &payload); err != nil {
		return nil, err
	}
	var out []connectors.CostRow
	for _, bucket := range payload.Data {
		start := time.Unix(bucket.StartTime, 0).UTC()
		end := time.Unix(bucket.EndTime, 0).UTC()
		for i, result := range bucket.Results {
			amount := result.Amount.Value
			if amount == "" {
				continue
			}
			currency := strings.ToUpper(result.Amount.Currency)
			if currency == "" {
				currency = "USD"
			}
			out = append(out, connectors.CostRow{
				ProviderName:      kind,
				BillingAccountID:  result.ProjectID,
				ServiceName:       result.LineItem,
				ServiceCategory:   "AI and Machine Learning",
				SKUID:             result.LineItem,
				ChargeCategory:    "Usage",
				ChargePeriodStart: start,
				ChargePeriodEnd:   end,
				BilledCost:        amount,
				EffectiveCost:     amount,
				BillingCurrency:   currency,
				UsageQuantity:     "1",
				UsageUnit:         "requests",
				SourceRecordID:    fmt.Sprintf("%s-%d-%d", raw.ID, bucket.StartTime, i),
			})
		}
	}
	return out, nil
}

func (c Connector) do(ctx context.Context, cred connectors.Credential, method, path string, body io.Reader) (*http.Response, error) {
	base := strings.TrimRight(c.base(), "/")
	request, err := http.NewRequestWithContext(ctx, method, base+path, body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(cred.Secret))
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	return c.client().Do(ctx, request)
}

func (c Connector) base() string {
	if strings.TrimSpace(c.Base) != "" {
		return c.Base
	}
	return "https://api.openai.com"
}

func (c Connector) client() httpclient.Client {
	if c.Client.HTTP != nil || c.Client.MaxAttempts != 0 {
		return c.Client
	}
	return httpclient.Client{HTTP: &http.Client{Timeout: 30 * time.Second}}
}

type costsResponse struct {
	Data []costBucket `json:"data"`
}

type costBucket struct {
	StartTime int64        `json:"start_time"`
	EndTime   int64        `json:"end_time"`
	Results   []costResult `json:"results"`
}

type costResult struct {
	LineItem  string     `json:"line_item"`
	ProjectID string     `json:"project_id"`
	Amount    costAmount `json:"amount"`
}

type costAmount struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

type vendorError struct {
	status     int
	retryAfter string
}

func (e *vendorError) Error() string { return "openai request failed" }

func (e *vendorError) Code() string {
	if e.status == http.StatusTooManyRequests {
		return "rate_limited"
	}
	return "vendor_unavailable"
}

func (e *vendorError) GetRetryAfter() time.Duration {
	if e.retryAfter == "" {
		return time.Second
	}
	seconds, err := time.ParseDuration(e.retryAfter + "s")
	if err != nil {
		return time.Second
	}
	return seconds
}
