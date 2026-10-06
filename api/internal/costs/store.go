package costs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/o-mid/contract-ops/api/internal/connectors"
)

const defaultLimit = 50
const maxLimit = 100

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Insert(ctx context.Context, workspaceID, connectionID, jobID, batchID string, rows []connectors.CostRow) error {
	if len(rows) == 0 {
		return nil
	}
	for _, row := range rows {
		id, err := newID("cost_")
		if err != nil {
			return err
		}
		_, err = s.pool.Exec(ctx, `
			INSERT INTO cost_rows (
				id, workspace_id, connection_id, job_id, batch_id,
				provider_name, billing_account_id, service_name, service_category, resource_id,
				sku_id, charge_category, charge_period_start, charge_period_end,
				billed_cost, effective_cost, billing_currency, usage_quantity, usage_unit, source_record_id
			) VALUES (
				$1, $2, $3, $4, $5,
				$6, $7, $8, $9, $10,
				$11, $12, $13, $14,
				$15, $16, $17, $18, $19, $20
			)
			ON CONFLICT (connection_id, source_record_id, charge_period_start) DO NOTHING`,
			id, workspaceID, connectionID, jobID, batchID,
			row.ProviderName, row.BillingAccountID, row.ServiceName, row.ServiceCategory, row.ResourceID,
			row.SKUID, row.ChargeCategory, row.ChargePeriodStart.UTC(), row.ChargePeriodEnd.UTC(),
			row.BilledCost, row.EffectiveCost, row.BillingCurrency, row.UsageQuantity, row.UsageUnit, row.SourceRecordID,
		)
		if err != nil {
			return fmt.Errorf("insert cost row: %w", err)
		}
	}
	return nil
}

func (s *Store) InsertTx(ctx context.Context, tx pgx.Tx, workspaceID, connectionID, jobID, batchID string, rows []connectors.CostRow) error {
	if len(rows) == 0 {
		return nil
	}
	for _, row := range rows {
		id, err := newID("cost_")
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO cost_rows (
				id, workspace_id, connection_id, job_id, batch_id,
				provider_name, billing_account_id, service_name, service_category, resource_id,
				sku_id, charge_category, charge_period_start, charge_period_end,
				billed_cost, effective_cost, billing_currency, usage_quantity, usage_unit, source_record_id
			) VALUES (
				$1, $2, $3, $4, $5,
				$6, $7, $8, $9, $10,
				$11, $12, $13, $14,
				$15, $16, $17, $18, $19, $20
			)
			ON CONFLICT (connection_id, source_record_id, charge_period_start) DO NOTHING`,
			id, workspaceID, connectionID, jobID, batchID,
			row.ProviderName, row.BillingAccountID, row.ServiceName, row.ServiceCategory, row.ResourceID,
			row.SKUID, row.ChargeCategory, row.ChargePeriodStart.UTC(), row.ChargePeriodEnd.UTC(),
			row.BilledCost, row.EffectiveCost, row.BillingCurrency, row.UsageQuantity, row.UsageUnit, row.SourceRecordID,
		)
		if err != nil {
			return fmt.Errorf("insert cost row: %w", err)
		}
	}
	return nil
}

func (s *Store) List(ctx context.Context, workspaceID string, query Query) (Page, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	cursorTime, cursorID, err := decodeCursor(query.Cursor)
	if err != nil {
		return Page{}, err
	}

	args := []any{workspaceID}
	where := `WHERE workspace_id = $1`
	n := 2
	if query.ConnectionID != "" {
		where += fmt.Sprintf(` AND connection_id = $%d`, n)
		args = append(args, query.ConnectionID)
		n++
	}
	if query.Start != nil {
		where += fmt.Sprintf(` AND charge_period_end >= $%d`, n)
		args = append(args, query.Start.UTC())
		n++
	}
	if query.End != nil {
		where += fmt.Sprintf(` AND charge_period_start <= $%d`, n)
		args = append(args, query.End.UTC())
		n++
	}
	if cursorTime != nil {
		where += fmt.Sprintf(` AND (charge_period_start, id) < ($%d, $%d)`, n, n+1)
		args = append(args, *cursorTime, cursorID)
		n += 2
	}

	var total int
	countSQL := `SELECT count(*) FROM cost_rows ` + where
	if err := s.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return Page{}, err
	}

	listSQL := `
		SELECT id, workspace_id, connection_id, job_id, batch_id,
		       provider_name, billing_account_id, service_name, service_category, resource_id,
		       sku_id, charge_category, charge_period_start, charge_period_end,
		       billed_cost, effective_cost, billing_currency, usage_quantity, usage_unit, source_record_id, ingested_at
		FROM cost_rows ` + where + `
		ORDER BY charge_period_start DESC, id DESC
		LIMIT ` + fmt.Sprint(limit+1)

	rows, err := s.pool.Query(ctx, listSQL, args...)
	if err != nil {
		return Page{}, err
	}
	defer rows.Close()

	page := Page{Total: total}
	var collected []Row
	for rows.Next() {
		var row Row
		if err := rows.Scan(
			&row.ID, &row.WorkspaceID, &row.ConnectionID, &row.JobID, &row.BatchID,
			&row.ProviderName, &row.BillingAccountID, &row.ServiceName, &row.ServiceCategory, &row.ResourceID,
			&row.SKUID, &row.ChargeCategory, &row.ChargePeriodStart, &row.ChargePeriodEnd,
			&row.BilledCost, &row.EffectiveCost, &row.BillingCurrency, &row.UsageQuantity, &row.UsageUnit, &row.SourceRecordID, &row.IngestedAt,
		); err != nil {
			return Page{}, err
		}
		collected = append(collected, row)
	}
	if err := rows.Err(); err != nil {
		return Page{}, err
	}
	if len(collected) > limit {
		tail := collected[limit]
		page.NextCursor = encodeCursor(tail.ChargePeriodStart, tail.ID)
		collected = collected[:limit]
	}
	page.Rows = collected
	return page, nil
}

func (s *Store) CountForConnection(ctx context.Context, connectionID string) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM cost_rows WHERE connection_id = $1`, connectionID).Scan(&n)
	return n, err
}

func encodeCursor(at time.Time, id string) string {
	return fmt.Sprintf("%s|%s", at.UTC().Format(time.RFC3339Nano), id)
}

func decodeCursor(raw string) (*time.Time, string, error) {
	if raw == "" {
		return nil, "", nil
	}
	parts := splitCursor(raw)
	if len(parts) != 2 {
		return nil, "", errors.New("invalid cursor")
	}
	parsed, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return nil, "", errors.New("invalid cursor")
	}
	return &parsed, parts[1], nil
}

func splitCursor(raw string) []string {
	for i := len(raw) - 1; i >= 0; i-- {
		if raw[i] == '|' {
			return []string{raw[:i], raw[i+1:]}
		}
	}
	return nil
}

func newID(prefix string) (string, error) {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(buf[:]), nil
}
