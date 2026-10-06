package costs

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/o-mid/contract-ops/api/internal/platform/auth"
	"github.com/o-mid/contract-ops/api/internal/platform/problem"
)

type Handler struct {
	store *Store
}

func NewHandler(store *Store) Handler {
	return Handler{store: store}
}

func (h Handler) Routes(router chi.Router) {
	router.Get("/v1/costs", h.list)
}

func (h Handler) list(writer http.ResponseWriter, request *http.Request) {
	workspaceID := auth.WorkspaceID(request.Context())
	if workspaceID == "" {
		problem.Write(writer, http.StatusUnauthorized, "auth_invalid", "missing bearer token")
		return
	}

	query := Query{
		ConnectionID: request.URL.Query().Get("connection_id"),
		Cursor:       request.URL.Query().Get("cursor"),
	}
	if raw := request.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			problem.Write(writer, http.StatusBadRequest, "", "limit must be a positive integer")
			return
		}
		query.Limit = n
	}
	if raw := request.URL.Query().Get("start"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			problem.Write(writer, http.StatusBadRequest, "", "start must be RFC3339")
			return
		}
		query.Start = &parsed
	}
	if raw := request.URL.Query().Get("end"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			problem.Write(writer, http.StatusBadRequest, "", "end must be RFC3339")
			return
		}
		query.End = &parsed
	}

	page, err := h.store.List(request.Context(), workspaceID, query)
	if err != nil {
		problem.Write(writer, http.StatusInternalServerError, "", "could not list costs")
		return
	}

	type rowJSON struct {
		ID                string `json:"id"`
		ConnectionID      string `json:"connectionId"`
		JobID             string `json:"jobId"`
		BatchID           string `json:"batchId"`
		ProviderName      string `json:"providerName"`
		BillingAccountID  string `json:"billingAccountId"`
		ServiceName       string `json:"serviceName"`
		ServiceCategory   string `json:"serviceCategory"`
		ResourceID        string `json:"resourceId"`
		SKUID             string `json:"skuId"`
		ChargeCategory    string `json:"chargeCategory"`
		ChargePeriodStart string `json:"chargePeriodStart"`
		ChargePeriodEnd   string `json:"chargePeriodEnd"`
		BilledCost        string `json:"billedCost"`
		EffectiveCost     string `json:"effectiveCost"`
		BillingCurrency   string `json:"billingCurrency"`
		UsageQuantity     string `json:"usageQuantity"`
		UsageUnit         string `json:"usageUnit"`
		SourceRecordID    string `json:"sourceRecordId"`
		IngestedAt        string `json:"ingestedAt"`
	}
	out := make([]rowJSON, 0, len(page.Rows))
	for _, row := range page.Rows {
		out = append(out, rowJSON{
			ID:                row.ID,
			ConnectionID:      row.ConnectionID,
			JobID:             row.JobID,
			BatchID:           row.BatchID,
			ProviderName:      row.ProviderName,
			BillingAccountID:  row.BillingAccountID,
			ServiceName:       row.ServiceName,
			ServiceCategory:   row.ServiceCategory,
			ResourceID:        row.ResourceID,
			SKUID:             row.SKUID,
			ChargeCategory:    row.ChargeCategory,
			ChargePeriodStart: row.ChargePeriodStart.UTC().Format(time.RFC3339Nano),
			ChargePeriodEnd:   row.ChargePeriodEnd.UTC().Format(time.RFC3339Nano),
			BilledCost:        row.BilledCost,
			EffectiveCost:     row.EffectiveCost,
			BillingCurrency:   row.BillingCurrency,
			UsageQuantity:     row.UsageQuantity,
			UsageUnit:         row.UsageUnit,
			SourceRecordID:    row.SourceRecordID,
			IngestedAt:        row.IngestedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	payload := map[string]any{
		"costs": out,
		"total": page.Total,
	}
	if page.NextCursor != "" {
		payload["nextCursor"] = page.NextCursor
	}
	writer.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(writer).Encode(payload)
}
