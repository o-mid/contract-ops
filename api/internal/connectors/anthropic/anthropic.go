package anthropic

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

const kind = "anthropic"

type Connector struct {
	Base   string
	Client httpclient.Client
}

func New() Connector {
	return Connector{Base: "https://api.anthropic.com"}
}

func (c Connector) Kind() string { return kind }

func (Connector) Describe() connectors.Descriptor {
	return connectors.Descriptor{
		Kind:        kind,
		DisplayName: "Anthropic",
		DocsURL:     "https://docs.anthropic.com/en/api/usage-cost",
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
		return fmt.Errorf("anthropic verify returned %d", response.StatusCode)
	}
	return nil
}

func (c Connector) Fetch(ctx context.Context, cred connectors.Credential, window connectors.Window, _ connectors.Cursor) (connectors.Page, error) {
	path := fmt.Sprintf(
		"/v1/organizations/cost_report?start_date=%s&end_date=%s",
		window.Start.UTC().Format("2006-01-02"),
		window.End.UTC().Format("2006-01-02"),
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
		return connectors.Page{}, fmt.Errorf("anthropic cost report returned %d", response.StatusCode)
	}
	if err != nil {
		return connectors.Page{}, err
	}
	return connectors.Page{
		Records: []connectors.RawRecord{{ID: "anthropic_costs", Payload: body}},
		Done:    true,
	}, nil
}

func (Connector) Normalize(raw connectors.RawRecord) ([]connectors.CostRow, error) {
	var payload costReport
	if err := json.Unmarshal(raw.Payload, &payload); err != nil {
		return nil, err
	}
	var out []connectors.CostRow
	for i, item := range payload.Data {
		start, err := time.Parse("2006-01-02", item.Date)
		if err != nil {
			return nil, err
		}
		end := start.Add(24 * time.Hour)
		currency := strings.ToUpper(item.Currency)
		if currency == "" {
			currency = "USD"
		}
		out = append(out, connectors.CostRow{
			ProviderName:      kind,
			BillingAccountID:  item.WorkspaceID,
			ServiceName:       item.Model,
			ServiceCategory:   "AI and Machine Learning",
			SKUID:             item.Model,
			ChargeCategory:    "Usage",
			ChargePeriodStart: start,
			ChargePeriodEnd:   end,
			BilledCost:        item.CostUSD,
			EffectiveCost:     item.CostUSD,
			BillingCurrency:   currency,
			UsageQuantity:     item.InputTokens,
			UsageUnit:         "tokens",
			SourceRecordID:    fmt.Sprintf("%s-%s-%d", raw.ID, item.Date, i),
		})
	}
	return out, nil
}

func (c Connector) do(ctx context.Context, cred connectors.Credential, method, path string, body io.Reader) (*http.Response, error) {
	base := strings.TrimRight(c.base(), "/")
	request, err := http.NewRequestWithContext(ctx, method, base+path, body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("x-api-key", strings.TrimSpace(cred.Secret))
	request.Header.Set("anthropic-version", "2023-06-01")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	return c.client().Do(ctx, request)
}

func (c Connector) base() string {
	if strings.TrimSpace(c.Base) != "" {
		return c.Base
	}
	return "https://api.anthropic.com"
}

func (c Connector) client() httpclient.Client {
	if c.Client.HTTP != nil || c.Client.MaxAttempts != 0 {
		return c.Client
	}
	return httpclient.Client{HTTP: &http.Client{Timeout: 30 * time.Second}}
}

type costReport struct {
	Data []costLine `json:"data"`
}

type costLine struct {
	Date         string `json:"date"`
	Model        string `json:"model"`
	WorkspaceID  string `json:"workspace_id"`
	CostUSD      string `json:"cost_usd"`
	Currency     string `json:"currency"`
	InputTokens  string `json:"input_tokens"`
	OutputTokens string `json:"output_tokens"`
}

type vendorError struct {
	status     int
	retryAfter string
}

func (e *vendorError) Error() string { return "anthropic request failed" }

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
