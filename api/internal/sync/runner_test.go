package sync

import (
	"bytes"
	"context"
	"database/sql"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/o-mid/contract-ops/api/internal/connections"
	"github.com/o-mid/contract-ops/api/internal/connectors"
	"github.com/o-mid/contract-ops/api/internal/connectors/fakevendor"
	"github.com/o-mid/contract-ops/api/internal/credentials"
	"github.com/o-mid/contract-ops/api/internal/platform/auth"
	"github.com/o-mid/contract-ops/api/internal/platform/db"
	"github.com/o-mid/contract-ops/api/internal/platform/migrate"
)

func TestThreeRunnersDoNotDuplicateAJob(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	connectionID := seedConnection(t, ctx, pool, "healthy")
	jobs := NewStore(pool)
	windowStart := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	if _, err := jobs.Enqueue(ctx, Job{
		ConnectionID: connectionID,
		WorkspaceID:  auth.LocalWorkspaceID,
		Kind:         "backfill",
		WindowStart:  windowStart,
		WindowEnd:    windowStart.Add(24 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}

	script := &scriptConnector{}
	runner := testRunner(t, pool, script)
	runFor(t, runner, 3, func() bool {
		n, err := jobs.BatchCount(ctx, connectionID)
		return err == nil && n == 1
	})

	n, err := jobs.BatchCount(ctx, connectionID)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("batches = %d, fetches = %d", n, script.fetches.Load())
	}
	if _, ok, err := jobs.Watermark(ctx, connectionID); err != nil || !ok {
		t.Fatalf("watermark ok=%v err=%v", ok, err)
	}
}

func TestDriftDoesNotAdvanceTheCursor(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	connectionID := seedConnection(t, ctx, pool, "healthy")
	jobs := NewStore(pool)
	start := time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)
	job, err := jobs.Enqueue(ctx, Job{
		ConnectionID: connectionID,
		WorkspaceID:  auth.LocalWorkspaceID,
		Kind:         "backfill",
		WindowStart:  start,
		WindowEnd:    start.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	claimed, ok, err := jobs.Claim(ctx, time.Second)
	if err != nil || !ok || claimed.ID != job.ID {
		t.Fatalf("claim = %+v %v %v", claimed, ok, err)
	}
	runner := testRunner(t, pool, driftConnector{})
	if err := runner.Execute(ctx, claimed); err != nil {
		t.Fatal(err)
	}
	stored, err := jobs.Get(ctx, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != "quarantined" || stored.ErrorCode != "schema_drift" {
		t.Fatalf("job = %+v", stored)
	}
	if _, ok, err := jobs.Watermark(ctx, connectionID); err != nil || ok {
		t.Fatalf("watermark moved: ok=%v err=%v", ok, err)
	}
	var status, reason string
	if err := pool.QueryRow(ctx, `SELECT status, status_reason_code FROM connections WHERE id = $1`, connectionID).Scan(&status, &reason); err != nil {
		t.Fatal(err)
	}
	if status != "degraded" || reason != "schema_drift" {
		t.Fatalf("connection = %s %s", status, reason)
	}
}

func TestBackfillChunksStaySerial(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	connectionID := seedConnection(t, ctx, pool, "healthy")
	jobs := NewStore(pool)
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	for _, chunk := range Chunks(start, start.Add(72*time.Hour)) {
		if _, err := jobs.Enqueue(ctx, Job{
			ConnectionID: connectionID,
			WorkspaceID:  auth.LocalWorkspaceID,
			Kind:         "backfill",
			WindowStart:  chunk[0],
			WindowEnd:    chunk[1],
		}); err != nil {
			t.Fatal(err)
		}
	}
	script := &scriptConnector{pause: 30 * time.Millisecond}
	runner := testRunner(t, pool, script)
	runFor(t, runner, 3, func() bool {
		n, err := jobs.BatchCount(ctx, connectionID)
		return err == nil && n == 3
	})
	if script.maxInflight.Load() != 1 {
		t.Fatalf("max in flight = %d", script.maxInflight.Load())
	}
	n, err := jobs.BatchCount(ctx, connectionID)
	if err != nil || n != 3 {
		t.Fatalf("batches = %d err=%v", n, err)
	}
}

func TestRateLimitRetriesWithoutASecondBatch(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	connectionID := seedConnection(t, ctx, pool, "healthy")
	jobs := NewStore(pool)
	start := time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC)
	job, err := jobs.Enqueue(ctx, Job{
		ConnectionID: connectionID,
		WorkspaceID:  auth.LocalWorkspaceID,
		Kind:         "scheduled",
		WindowStart:  start,
		WindowEnd:    start.Add(time.Hour),
		MaxAttempts:  3,
	})
	if err != nil {
		t.Fatal(err)
	}
	script := &scriptConnector{failures: 1}
	runner := testRunner(t, pool, script)
	runner.Lease = time.Second
	claimed, ok, err := jobs.Claim(ctx, time.Second)
	if err != nil || !ok {
		t.Fatal(err)
	}
	if err := runner.Execute(ctx, claimed); err != nil {
		t.Fatal(err)
	}
	stored, err := jobs.Get(ctx, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != "queued" || stored.ErrorCode != "rate_limited" {
		t.Fatalf("after limit = %+v", stored)
	}
	if _, err := pool.Exec(ctx, `UPDATE sync_jobs SET next_run_at = now() WHERE id = $1`, job.ID); err != nil {
		t.Fatal(err)
	}
	claimed, ok, err = jobs.Claim(ctx, time.Second)
	if err != nil || !ok {
		t.Fatalf("second claim %v %v", ok, err)
	}
	if err := runner.Execute(ctx, claimed); err != nil {
		t.Fatal(err)
	}
	stored, err = jobs.Get(ctx, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != "succeeded" {
		t.Fatalf("status = %s", stored.Status)
	}
	n, err := jobs.BatchCount(ctx, connectionID)
	if err != nil || n != 1 {
		t.Fatalf("batches = %d err=%v", n, err)
	}
}

func testRunner(t *testing.T, pool *pgxpool.Pool, connector connectors.Connector) *Runner {
	t.Helper()
	key, err := credentials.NewEnvKey(bytes.Repeat([]byte{4}, 32))
	if err != nil {
		t.Fatal(err)
	}
	return &Runner{
		Jobs:        NewStore(pool),
		Connections: connections.NewStore(pool),
		Sealer:      credentials.NewSealer(key),
		Registry:    connectors.NewRegistry(connector),
		Lease:       time.Second,
	}
}

func runFor(t *testing.T, runner *Runner, workers int, done func() bool) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var group sync.WaitGroup
	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			_ = runner.Run(ctx)
		}()
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if done() {
			cancel()
			group.Wait()
			return
		}
		time.Sleep(15 * time.Millisecond)
	}
	cancel()
	group.Wait()
	t.Fatal("timed out waiting for runners")
}

type scriptConnector struct {
	failures    int
	pause       time.Duration
	fetches     atomic.Int32
	inflight    atomic.Int32
	maxInflight atomic.Int32
}

func (s *scriptConnector) Kind() string { return "fakevendor" }

func (s *scriptConnector) Describe() connectors.Descriptor {
	return connectors.Descriptor{Kind: "fakevendor"}
}

func (s *scriptConnector) Verify(context.Context, connectors.Credential) error { return nil }

func (s *scriptConnector) Fetch(_ context.Context, _ connectors.Credential, window connectors.Window, _ connectors.Cursor) (connectors.Page, error) {
	current := s.inflight.Add(1)
	for {
		max := s.maxInflight.Load()
		if current <= max || s.maxInflight.CompareAndSwap(max, current) {
			break
		}
	}
	if s.pause > 0 {
		time.Sleep(s.pause)
	}
	s.inflight.Add(-1)
	n := s.fetches.Add(1)
	if int(n) <= s.failures {
		return connectors.Page{}, &fakevendor.VendorError{Status: 429, RetryAfter: time.Millisecond}
	}
	payload := []byte(`{"id":"fv_1","service":"completions","sku":"tokens","start":"2026-07-01T00:00:00Z","end":"2026-07-02T00:00:00Z","billed":"1.25","currency":"USD","quantity":"1","unit":"tokens","window":"` + window.Start.Format(time.RFC3339) + `"}`)
	return connectors.Page{Records: []connectors.RawRecord{{ID: "fv_1", Payload: payload}}, Done: true}, nil
}

func (s *scriptConnector) Normalize(raw connectors.RawRecord) ([]connectors.CostRow, error) {
	return fakevendor.New().Normalize(raw)
}

type driftConnector struct{}

func (driftConnector) Kind() string { return "fakevendor" }

func (driftConnector) Describe() connectors.Descriptor {
	return connectors.Descriptor{Kind: "fakevendor"}
}

func (driftConnector) Verify(context.Context, connectors.Credential) error { return nil }

func (driftConnector) Fetch(context.Context, connectors.Credential, connectors.Window, connectors.Cursor) (connectors.Page, error) {
	return connectors.Page{Records: []connectors.RawRecord{{ID: "bad", Payload: []byte(`{"id":"bad","service_v2":"completions"}`)}}, Done: true}, nil
}

func (driftConnector) Normalize(raw connectors.RawRecord) ([]connectors.CostRow, error) {
	return fakevendor.New().Normalize(raw)
}

func seedConnection(t *testing.T, ctx context.Context, pool *pgxpool.Pool, status string) string {
	t.Helper()
	if _, err := pool.Exec(ctx, `INSERT INTO workspaces (id, name) VALUES ($1, 'Local') ON CONFLICT (id) DO NOTHING`, auth.LocalWorkspaceID); err != nil {
		t.Fatal(err)
	}
	key, err := credentials.NewEnvKey(bytes.Repeat([]byte{4}, 32))
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := credentials.NewSealer(key).Seal(ctx, []byte("demo-secret-value"))
	if err != nil {
		t.Fatal(err)
	}
	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO connections (id, workspace_id, kind, name, status)
		VALUES ('conn_sync_test_' || substr(md5(random()::text), 1, 8), $1, 'fakevendor', 'Demo', $2)
		RETURNING id`, auth.LocalWorkspaceID, status).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO credentials (id, connection_id, ciphertext, wrapped_key, key_version, fingerprint)
		VALUES ('cred_' || $1, $1, $2, $3, $4, $5)`,
		id, sealed.Ciphertext, sealed.WrappedKey, sealed.KeyVersion, sealed.Fingerprint); err != nil {
		t.Fatal(err)
	}
	return id
}

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	admin, err := sql.Open("pgx", base)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = admin.ExecContext(context.Background(), `DROP DATABASE IF EXISTS contract_ops_sync_test WITH (FORCE)`)
		_ = admin.Close()
	})
	if err := admin.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	_, _ = admin.ExecContext(ctx, `DROP DATABASE IF EXISTS contract_ops_sync_test WITH (FORCE)`)
	if _, err := admin.ExecContext(ctx, `CREATE DATABASE contract_ops_sync_test`); err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(base)
	if err != nil {
		t.Fatal(err)
	}
	parsed.Path = "/contract_ops_sync_test"
	sqlDB, err := sql.Open("pgx", parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := migrate.Up(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}
	pool, err := db.Open(ctx, parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}
