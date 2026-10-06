package sync

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const schedulerLock = "contract-ops-sync-scheduler"

type Scheduler struct {
	Store    *Store
	Pool     *pgxpool.Pool
	Interval time.Duration
	Lookback time.Duration
}

// Run holds a session advisory lock. The process that gets the lock is the
// only one that enqueues scheduled jobs. The lock drops when the connection
// closes, including after a crash.
func (s *Scheduler) Run(ctx context.Context) error {
	conn, err := s.Pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	var locked bool
	if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock(hashtext($1)::bigint)`, schedulerLock).Scan(&locked); err != nil {
		return err
	}
	if !locked {
		<-ctx.Done()
		return ctx.Err()
	}
	defer func() {
		_, _ = conn.Exec(context.Background(), `SELECT pg_advisory_unlock(hashtext($1)::bigint)`, schedulerLock)
	}()

	interval := s.Interval
	if interval <= 0 {
		interval = time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if _, err := s.Store.EnqueueScheduled(ctx, s.Lookback); err != nil && ctx.Err() == nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
