package openai

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
		case "/v1/organization/costs":
			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write([]byte(`{
				"data":[{"start_time":1704067200,"end_time":1704153600,"results":[
					{"line_item":"gpt-4","project_id":"proj_1","amount":{"value":"2.50","currency":"usd"}}
				]}]
			}`))
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	conn := Connector{Base: server.URL}
	if err := conn.Verify(context.Background(), connectors.Credential{Secret: "sk-test"}); err != nil {
		t.Fatal(err)
	}
	page, err := conn.Fetch(context.Background(), connectors.Credential{Secret: "sk-test"}, connectors.Window{
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
	if len(rows) != 1 || rows[0].BilledCost != "2.50" || rows[0].ProviderName != "openai" {
		t.Fatalf("rows = %+v", rows)
	}
}
