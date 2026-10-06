package sync

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/o-mid/contract-ops/api/internal/connections"
	"github.com/o-mid/contract-ops/api/internal/connectors"
	"github.com/o-mid/contract-ops/api/internal/costs"
	"github.com/o-mid/contract-ops/api/internal/events"
)

var ErrNotFound = errors.New("job not found")

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Enqueue(ctx context.Context, job Job) (Job, error) {
	if job.ID == "" {
		id, err := newID("job_")
		if err != nil {
			return Job{}, err
		}
		job.ID = id
	}
	if job.MaxAttempts == 0 {
		job.MaxAttempts = 5
	}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO sync_jobs (
			id, connection_id, workspace_id, kind, window_start, window_end, status, max_attempts
		) VALUES ($1, $2, $3, $4, $5, $6, 'queued', $7)
		ON CONFLICT DO NOTHING
		RETURNING id`,
		job.ID, job.ConnectionID, job.WorkspaceID, job.Kind, job.WindowStart, job.WindowEnd, job.MaxAttempts,
	).Scan(&job.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		if job.Kind == "scheduled" {
			return s.activeScheduled(ctx, job.ConnectionID)
		}
		return s.jobByWindow(ctx, job)
	}
	if err != nil {
		return Job{}, err
	}
	job.Status = "queued"
	return job, nil
}

func (s *Store) activeScheduled(ctx context.Context, connectionID string) (Job, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, connection_id, workspace_id, kind, window_start, window_end, status, attempt, max_attempts, error_code, error_detail
		FROM sync_jobs
		WHERE connection_id = $1 AND kind = 'scheduled' AND status IN ('queued', 'running')
		ORDER BY created_at DESC
		LIMIT 1`, connectionID)
	return scanJob(row)
}

func (s *Store) jobByWindow(ctx context.Context, job Job) (Job, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, connection_id, workspace_id, kind, window_start, window_end, status, attempt, max_attempts, error_code, error_detail
		FROM sync_jobs
		WHERE connection_id = $1 AND kind = $2 AND window_start = $3 AND window_end = $4
		ORDER BY created_at DESC
		LIMIT 1`, job.ConnectionID, job.Kind, job.WindowStart, job.WindowEnd)
	return scanJob(row)
}

func (s *Store) EnqueueScheduled(ctx context.Context, lookback time.Duration) (int, error) {
	if lookback <= 0 {
		lookback = 24 * time.Hour
	}
	rows, err := s.pool.Query(ctx, `
		SELECT c.id, c.workspace_id, COALESCE(cur.high_watermark, now() - $1::interval)
		FROM connections c
		LEFT JOIN sync_cursors cur ON cur.connection_id = c.id AND cur.stream = $2
		WHERE c.status = 'healthy'`, lookback.String(), streamUsage)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var n int
	for rows.Next() {
		var connectionID, workspaceID string
		var watermark time.Time
		if err := rows.Scan(&connectionID, &workspaceID, &watermark); err != nil {
			return n, err
		}
		end := time.Now().UTC()
		start := watermark.Add(-lookback)
		if _, err := s.Enqueue(ctx, Job{
			ConnectionID: connectionID,
			WorkspaceID:  workspaceID,
			Kind:         "scheduled",
			WindowStart:  start,
			WindowEnd:    end,
		}); err != nil {
			return n, err
		}
		n++
	}
	return n, rows.Err()
}

// Claim locks one due job with SKIP LOCKED, then takes a transaction advisory
// lock on the connection. A second live lease on that connection is left for
// its owner. A running row whose lease_until is already past can be taken.
func (s *Store) Claim(ctx context.Context, lease time.Duration) (Job, bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Job{}, false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var connectionID, id string
	err = tx.QueryRow(ctx, `
		SELECT id, connection_id
		FROM sync_jobs
		WHERE next_run_at <= now()
		  AND (
		    status = 'queued'
		    OR (status = 'running' AND lease_until < now())
		  )
		ORDER BY next_run_at, id
		FOR UPDATE SKIP LOCKED
		LIMIT 1`).Scan(&id, &connectionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Job{}, false, nil
	}
	if err != nil {
		return Job{}, false, err
	}

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1)::bigint)`, connectionID); err != nil {
		return Job{}, false, err
	}
	var busy bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM sync_jobs
			WHERE connection_id = $1 AND status = 'running' AND lease_until >= now() AND id <> $2
		)`, connectionID, id).Scan(&busy); err != nil {
		return Job{}, false, err
	}
	if busy {
		if err := tx.Commit(ctx); err != nil {
			return Job{}, false, err
		}
		return Job{}, false, nil
	}

	if _, err := tx.Exec(ctx, `
		UPDATE sync_jobs
		SET status = 'running',
		    attempt = attempt + 1,
		    lease_until = now() + $2::interval,
		    heartbeat_at = now()
		WHERE id = $1`, id, lease.String()); err != nil {
		return Job{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Job{}, false, err
	}
	job, err := s.Get(ctx, id)
	if err != nil {
		return Job{}, false, err
	}
	return job, true, nil
}

func (s *Store) Heartbeat(ctx context.Context, id string, lease time.Duration) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE sync_jobs
		SET heartbeat_at = now(), lease_until = now() + $2::interval
		WHERE id = $1 AND status = 'running'`, id, lease.String())
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("lost lease for %s", id)
	}
	return nil
}

func (s *Store) Get(ctx context.Context, id string) (Job, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, connection_id, workspace_id, kind, window_start, window_end, status, attempt, max_attempts, error_code, error_detail
		FROM sync_jobs WHERE id = $1`, id)
	job, err := scanJob(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Job{}, ErrNotFound
	}
	return job, err
}

func (s *Store) Status(ctx context.Context, id string) (string, error) {
	var status string
	err := s.pool.QueryRow(ctx, `SELECT status FROM sync_jobs WHERE id = $1`, id).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return status, err
}

func (s *Store) Cancel(ctx context.Context, workspaceID, id string) (Job, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE sync_jobs
		SET status = 'cancelled', lease_until = NULL
		WHERE id = $1 AND workspace_id = $2 AND status IN ('queued', 'running')`, id, workspaceID)
	if err != nil {
		return Job{}, err
	}
	if tag.RowsAffected() == 0 {
		job, err := s.Get(ctx, id)
		if err != nil {
			return Job{}, err
		}
		if job.WorkspaceID != workspaceID {
			return Job{}, ErrNotFound
		}
		return job, nil
	}
	return s.Get(ctx, id)
}

func (s *Store) CommitPage(ctx context.Context, job Job, batch Batch, watermark time.Time, costRows []connectors.CostRow) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	batchID, err := newID("batch_")
	if err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `
		INSERT INTO raw_batches (id, job_id, connection_id, record_count, payload_hash, quarantined, drift_report)
		VALUES ($1, $2, $3, $4, $5, false, NULL)
		ON CONFLICT (connection_id, payload_hash) DO NOTHING`,
		batchID, job.ID, job.ConnectionID, batch.RecordCount, batch.Hash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		if err := tx.QueryRow(ctx, `
			SELECT id FROM raw_batches WHERE connection_id = $1 AND payload_hash = $2`,
			job.ConnectionID, batch.Hash).Scan(&batchID); err != nil {
			return err
		}
	}
	costStore := costs.NewStore(s.pool)
	if err := costStore.InsertTx(ctx, tx, job.WorkspaceID, job.ConnectionID, job.ID, batchID, costRows); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO sync_cursors (connection_id, stream, cursor, high_watermark, updated_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (connection_id, stream) DO UPDATE
		SET cursor = EXCLUDED.cursor,
		    high_watermark = GREATEST(sync_cursors.high_watermark, EXCLUDED.high_watermark),
		    updated_at = now()`,
		job.ConnectionID, streamUsage, watermark.UTC().Format(time.RFC3339Nano), watermark.UTC()); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE sync_jobs
		SET status = 'succeeded', lease_until = NULL, error_code = '', error_detail = ''
		WHERE id = $1 AND status = 'running'`, job.ID); err != nil {
		return err
	}
	if err := setConnectionStatus(ctx, tx, job.ConnectionID, connections.StatusHealthy, "", true); err != nil {
		return err
	}
	if err := insertEvent(ctx, tx, job.ConnectionID, "sync.succeeded"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Quarantine records the batch and marks the job quarantined.
// It does not update sync_cursors and it does not call Retry.
func (s *Store) Quarantine(ctx context.Context, job Job, batch Batch) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	batchID, err := newID("batch_")
	if err != nil {
		return err
	}
	report := fmt.Sprintf(`{"path":%q}`, batch.DriftPath)
	if _, err := tx.Exec(ctx, `
		INSERT INTO raw_batches (id, job_id, connection_id, record_count, payload_hash, quarantined, drift_report)
		VALUES ($1, $2, $3, $4, $5, true, $6::jsonb)
		ON CONFLICT (connection_id, payload_hash) DO NOTHING`,
		batchID, job.ID, job.ConnectionID, batch.RecordCount, batch.Hash, report); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE sync_jobs
		SET status = 'quarantined', lease_until = NULL, error_code = 'schema_drift', error_detail = $2
		WHERE id = $1 AND status = 'running'`, job.ID, "schema drift at "+batch.DriftPath); err != nil {
		return err
	}
	if err := setConnectionStatus(ctx, tx, job.ConnectionID, connections.StatusDegraded, "schema_drift", false); err != nil {
		return err
	}
	if err := insertEvent(ctx, tx, job.ConnectionID, "sync.quarantined"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) Retry(ctx context.Context, job Job, code, detail string, after time.Duration) error {
	if job.Attempt >= job.MaxAttempts {
		return s.Fail(ctx, job, code, detail)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `
		UPDATE sync_jobs
		SET status = 'queued', lease_until = NULL, next_run_at = now() + $2::interval, error_code = $3, error_detail = $4
		WHERE id = $1 AND status = 'running'`, job.ID, after.String(), code, detail); err != nil {
		return err
	}
	next := connections.Apply(connections.StatusHealthy, code)
	if err := setConnectionStatus(ctx, tx, job.ConnectionID, next, code, false); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Fail maps code through Apply starting from healthy, then writes that status.
// The update skips a paused connection, so pause still wins.
func (s *Store) Fail(ctx context.Context, job Job, code, detail string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `
		UPDATE sync_jobs
		SET status = 'failed', lease_until = NULL, error_code = $2, error_detail = $3
		WHERE id = $1 AND status = 'running'`, job.ID, code, detail); err != nil {
		return err
	}
	next := connections.Apply(connections.StatusHealthy, code)
	if err := setConnectionStatus(ctx, tx, job.ConnectionID, next, code, false); err != nil {
		return err
	}
	if err := insertEvent(ctx, tx, job.ConnectionID, "sync.failed"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) Connection(ctx context.Context, id string) (string, string, error) {
	var kind, status string
	err := s.pool.QueryRow(ctx, `SELECT kind, status FROM connections WHERE id = $1`, id).Scan(&kind, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrNotFound
	}
	return kind, status, err
}

func (s *Store) Watermark(ctx context.Context, connectionID string) (time.Time, bool, error) {
	var watermark *time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT high_watermark FROM sync_cursors WHERE connection_id = $1 AND stream = $2`,
		connectionID, streamUsage).Scan(&watermark)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, false, nil
	}
	if err != nil || watermark == nil {
		return time.Time{}, false, err
	}
	return watermark.UTC(), true, nil
}

func (s *Store) BatchCount(ctx context.Context, connectionID string) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM raw_batches WHERE connection_id = $1`, connectionID).Scan(&n)
	return n, err
}

type scannable interface {
	Scan(dest ...any) error
}

func scanJob(row scannable) (Job, error) {
	var job Job
	err := row.Scan(&job.ID, &job.ConnectionID, &job.WorkspaceID, &job.Kind, &job.WindowStart, &job.WindowEnd, &job.Status, &job.Attempt, &job.MaxAttempts, &job.ErrorCode, &job.ErrorDetail)
	return job, err
}

func setConnectionStatus(ctx context.Context, tx pgx.Tx, connectionID string, status connections.Status, reason string, success bool) error {
	now := time.Now().UTC()
	var successAt, errorAt any
	if success {
		successAt = now
	}
	if reason != "" {
		errorAt = now
	}
	_, err := tx.Exec(ctx, `
		UPDATE connections
		SET status = $2,
		    status_reason_code = $3,
		    last_success_at = COALESCE($4, last_success_at),
		    last_error_at = COALESCE($5, last_error_at),
		    updated_at = $6
		WHERE id = $1 AND status <> 'paused'`,
		connectionID, string(status), reason, successAt, errorAt, now)
	return err
}

func insertEvent(ctx context.Context, tx pgx.Tx, connectionID, eventType string) error {
	id, err := newID("evt_")
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO activity_events (id, source, type, status, occurred_at, correlation_id, connection_id)
		VALUES ($1, 'sync', $2, $3, now(), $4, $4)`,
		id, eventType, string(events.StatusProcessed), connectionID)
	return err
}

func newID(prefix string) (string, error) {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(buf[:]), nil
}
