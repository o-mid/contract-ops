package sync

import (
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"time"

	"github.com/o-mid/contract-ops/api/internal/connections"
	"github.com/o-mid/contract-ops/api/internal/connectors"
	"github.com/o-mid/contract-ops/api/internal/credentials"
)

type Runner struct {
	Jobs        *Store
	Connections *connections.Store
	Sealer      *credentials.Sealer
	Registry    *connectors.Registry
	Lease       time.Duration
	Lookback    time.Duration
}

func (r *Runner) Run(ctx context.Context) error {
	lease := r.Lease
	if lease <= 0 {
		lease = 30 * time.Second
	}
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		job, ok, err := r.Jobs.Claim(ctx, lease)
		if err != nil {
			return err
		}
		if !ok {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(20 * time.Millisecond):
			}
			continue
		}
		if err := r.Execute(ctx, job); err != nil {
			return err
		}
	}
}

// Execute opens the active credential and fetches one window.
// Schema drift quarantines the batch and leaves the cursor where it was.
// Successful pages persist normalized cost rows with the committed batch.
func (r *Runner) Execute(ctx context.Context, job Job) error {
	kind, status, err := r.Jobs.Connection(ctx, job.ConnectionID)
	if err != nil {
		return err
	}
	if status == string(connections.StatusPaused) {
		_, err := r.Jobs.Cancel(ctx, job.WorkspaceID, job.ID)
		return err
	}
	if stopped, err := r.stopped(ctx, job.ID); err != nil || stopped {
		return err
	}

	_, sealed, err := r.Connections.ActiveSecret(ctx, job.WorkspaceID, job.ConnectionID)
	if err != nil {
		return r.Jobs.Fail(ctx, job, "auth_invalid", "credential is missing")
	}
	plain, err := r.Sealer.Open(ctx, sealed)
	if err != nil {
		return r.Jobs.Fail(ctx, job, "auth_invalid", "credential could not be opened")
	}
	connector, ok := r.Registry.Get(kind)
	if !ok {
		return r.Jobs.Fail(ctx, job, "", "connector is not enabled")
	}

	page, err := connector.Fetch(ctx, connectors.Credential{Secret: string(plain)}, connectors.Window{
		Start: job.WindowStart,
		End:   job.WindowEnd,
	}, connectors.Cursor{})
	if err != nil {
		return r.failFetch(ctx, job, err)
	}
	if stopped, err := r.stopped(ctx, job.ID); err != nil || stopped {
		return err
	}

	var costRows []connectors.CostRow
	for _, record := range page.Records {
		rows, err := connector.Normalize(record)
		if err != nil {
			var drift interface{ SchemaDrift() string }
			if errors.As(err, &drift) {
				return r.Jobs.Quarantine(ctx, job, Batch{
					RecordCount: len(page.Records),
					Hash:        hashPage(job, page),
					DriftPath:   drift.SchemaDrift(),
				})
			}
			return r.failFetch(ctx, job, err)
		}
		costRows = append(costRows, rows...)
	}

	return r.Jobs.CommitPage(ctx, job, Batch{
		RecordCount: len(page.Records),
		Hash:        hashPage(job, page),
	}, job.WindowEnd, costRows)
}

func (r *Runner) failFetch(ctx context.Context, job Job, err error) error {
	if errors.Is(err, connectors.ErrRejected) {
		return r.Jobs.Fail(ctx, job, "auth_invalid", "credential was rejected")
	}
	var limited interface {
		Code() string
		GetRetryAfter() time.Duration
	}
	if errors.As(err, &limited) {
		wait := backoff(job.Attempt)
		if after := limited.GetRetryAfter(); after > wait {
			wait = after
		}
		return r.Jobs.Retry(ctx, job, limited.Code(), err.Error(), wait)
	}
	return r.Jobs.Retry(ctx, job, "vendor_unavailable", err.Error(), backoff(job.Attempt))
}

func (r *Runner) stopped(ctx context.Context, id string) (bool, error) {
	status, err := r.Jobs.Status(ctx, id)
	if err != nil {
		return false, err
	}
	return status == "cancelled", nil
}

func hashPage(job Job, page connectors.Page) []byte {
	sum := sha256.New()
	_, _ = io.WriteString(sum, job.ConnectionID)
	_, _ = sum.Write([]byte{0})
	_, _ = io.WriteString(sum, job.WindowStart.UTC().Format(time.RFC3339Nano))
	_, _ = sum.Write([]byte{0})
	_, _ = io.WriteString(sum, job.WindowEnd.UTC().Format(time.RFC3339Nano))
	for _, record := range page.Records {
		_, _ = sum.Write([]byte{0})
		_, _ = sum.Write(record.Payload)
	}
	return sum.Sum(nil)
}
