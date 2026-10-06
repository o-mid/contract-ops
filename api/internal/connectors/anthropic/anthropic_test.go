package anthropic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/o-mid/contract-ops/api/internal/connectors"
)

func TestVerifyAndNormalizeCosts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/v1/models":
			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write([]byte(`{"data":[]}`))
		case "/v1/organizations/cost_report":
			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write([]byte(`{
				"data":[{"date":"2024-01-01","model":"claude-3-5-sonnet","workspace_id":"ws_1","cost_usd":"1.10","currency":"usd","input_tokens":"1000","output_tokens":"200"}]
			}`))
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	conn := Connector{Base: server.URL}
	if err := conn.Verify(context.Background(), connectors.Credential{Secret: "sk-ant-test"}); err != nil {
		t.Fatal(err)
	}
	page, err := conn.Fetch(context.Background(), connectors.Credential{Secret: "sk-ant-test"}, connectors.Window{
		Start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
	}, connectors.Cursor{})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := conn.Normalize(page.Records[0])
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].BilledCost != "1.10" || rows[0].ProviderName != "anthropic" {
		t.Fatalf("rows = %+v", rows)
	}
}
