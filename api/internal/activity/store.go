// Package activity stores the event feed in Postgres.
package activity

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/o-mid/contract-ops/api/internal/events"
)

// Store is the Postgres-backed event feed. Subscribers in this process are
// woken when this process inserts a row. Cross-process wakeups are added
// with LISTEN/NOTIFY.
type Store struct {
	pool *pgxpool.Pool
	mu   sync.Mutex
	subs map[chan struct{}]struct{}
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{
		pool: pool,
		subs: make(map[chan struct{}]struct{}),
	}
}

func (s *Store) Len(ctx context.Context) (int, error) {
	var count int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM activity_events`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count events: %w", err)
	}
	return count, nil
}

func (s *Store) List(ctx context.Context, query events.Query) (events.Page, error) {
	limit := query.Limit
	if limit == 0 {
		limit = events.DefaultLimit
	}
	if limit < 1 || limit > events.MaxLimit {
		limit = events.DefaultLimit
	}

	var cursorTime *time.Time
	cursorID := ""
	if query.Cursor != "" {
		parsed, id, err := events.DecodeCursor(query.Cursor)
		if err != nil {
			return events.Page{}, err
		}
		cursorTime = &parsed
		cursorID = id
	}

	status := query.Status
	if status == "all" {
		status = ""
	}
	text := strings.TrimSpace(query.Text)

	const countSQL = `
		SELECT count(*)
		FROM activity_events
		WHERE ($1 = '' OR status = $1)
		  AND ($2 = '' OR connection_id = $2)
		  AND (
		    $3 = ''
		    OR strpos(lower(id), lower($3)) > 0
		    OR strpos(lower(source), lower($3)) > 0
		    OR strpos(lower(type), lower($3)) > 0
		    OR strpos(lower(correlation_id), lower($3)) > 0
		  )`

	var total int
	if err := s.pool.QueryRow(ctx, countSQL, status, query.ConnectionID, text).Scan(&total); err != nil {
		return events.Page{}, fmt.Errorf("count filtered events: %w", err)
	}

	const listSQL = `
		SELECT id, source, type, status, occurred_at, correlation_id, COALESCE(connection_id, '')
		FROM activity_events
		WHERE ($1 = '' OR status = $1)
		  AND ($2 = '' OR connection_id = $2)
		  AND (
		    $3 = ''
		    OR strpos(lower(id), lower($3)) > 0
		    OR strpos(lower(source), lower($3)) > 0
		    OR strpos(lower(type), lower($3)) > 0
		    OR strpos(lower(correlation_id), lower($3)) > 0
		  )
		  AND (
		    $4::timestamptz IS NULL
		    OR (occurred_at, id) < ($4::timestamptz, $5::text)
		  )
		ORDER BY occurred_at DESC, id DESC
		LIMIT $6`

	rows, err := s.pool.Query(ctx, listSQL, status, query.ConnectionID, text, cursorTime, cursorID, limit)
	if err != nil {
		return events.Page{}, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	found := make([]events.Event, 0)
	for rows.Next() {
		var event events.Event
		var statusValue string
		if err := rows.Scan(
			&event.ID,
			&event.Source,
			&event.Type,
			&statusValue,
			&event.OccurredAt,
			&event.CorrelationID,
			&event.ConnectionID,
		); err != nil {
			return events.Page{}, fmt.Errorf("scan event: %w", err)
		}
		event.Status = events.Status(statusValue)
		event.OccurredAt = event.OccurredAt.UTC()
		found = append(found, event)
	}
	if err := rows.Err(); err != nil {
		return events.Page{}, fmt.Errorf("list events: %w", err)
	}

	page := events.Page{Events: found, Total: total}
	if len(found) == limit {
		last := found[len(found)-1]
		var remaining int
		if err := s.pool.QueryRow(ctx, `
			SELECT count(*)
			FROM activity_events
			WHERE ($1 = '' OR status = $1)
			  AND ($2 = '' OR connection_id = $2)
			  AND (
			    $3 = ''
			    OR strpos(lower(id), lower($3)) > 0
			    OR strpos(lower(source), lower($3)) > 0
			    OR strpos(lower(type), lower($3)) > 0
			    OR strpos(lower(correlation_id), lower($3)) > 0
			  )
			  AND (occurred_at, id) < ($4::timestamptz, $5::text)`,
			status, query.ConnectionID, text, last.OccurredAt, last.ID,
		).Scan(&remaining); err != nil {
			return events.Page{}, fmt.Errorf("count remaining events: %w", err)
		}
		if remaining > 0 {
			page.NextCursor = events.EncodeCursor(last.OccurredAt, last.ID)
		}
	}

	return page, nil
}

func (s *Store) Append(ctx context.Context, event events.Event) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		INSERT INTO activity_events (id, source, type, status, occurred_at, correlation_id, connection_id)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''))
		ON CONFLICT (id) DO NOTHING`,
		event.ID,
		event.Source,
		event.Type,
		string(event.Status),
		event.OccurredAt.UTC(),
		event.CorrelationID,
		event.ConnectionID,
	)
	if err != nil {
		return false, fmt.Errorf("insert event: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}
	s.notify()
	return true, nil
}

func (s *Store) Subscribe(context.Context) (<-chan struct{}, func(), error) {
	ch := make(chan struct{}, 1)

	s.mu.Lock()
	s.subs[ch] = struct{}{}
	s.mu.Unlock()

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			s.mu.Lock()
			if _, ok := s.subs[ch]; ok {
				delete(s.subs, ch)
				close(ch)
			}
			s.mu.Unlock()
		})
	}

	return ch, cancel, nil
}

func (s *Store) notify() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range s.subs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}
