package events

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"
)

const MaxEvents = 8

const (
	DefaultLimit = 50
	MaxLimit     = 100
)

type Status string

const (
	StatusProcessed Status = "processed"
	StatusPending   Status = "pending"
	StatusFailed    Status = "failed"
)

type Event struct {
	ID            string    `json:"id"`
	Source        string    `json:"source"`
	Type          string    `json:"type"`
	Status        Status    `json:"status"`
	OccurredAt    time.Time `json:"occurredAt"`
	CorrelationID string    `json:"correlationId"`
	ConnectionID  string    `json:"connectionId,omitempty"`
}

type Page struct {
	Events     []Event `json:"events"`
	Total      int     `json:"total"`
	NextCursor string  `json:"nextCursor,omitempty"`
}

// Query is a filtered, keyset-paginated read of the feed.
type Query struct {
	Text         string
	Status       string
	ConnectionID string
	Cursor       string
	Limit        int
}

// Feed is the event list used by the HTTP handlers. The memory store backs
// tests. The Postgres store backs the running API.
type Feed interface {
	List(ctx context.Context, query Query) (Page, error)
	Append(ctx context.Context, event Event) (bool, error)
	Len(ctx context.Context) (int, error)
	Subscribe(ctx context.Context) (<-chan struct{}, func(), error)
}

// Store is the in-memory feed. Unit tests use it. The running API uses activity.Store.
type Store struct {
	mu     sync.RWMutex
	events []Event
	subs   map[chan struct{}]struct{}
}

func NewStore(events []Event) *Store {
	copyOfEvents := append([]Event(nil), events...)
	sort.Slice(copyOfEvents, func(i, j int) bool {
		return newer(copyOfEvents[i], copyOfEvents[j])
	})

	return &Store{
		events: copyOfEvents,
		subs:   make(map[chan struct{}]struct{}),
	}
}

func (s *Store) Len(context.Context) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.events), nil
}

func (s *Store) List(_ context.Context, query Query) (Page, error) {
	limit, err := normalizeLimit(query.Limit)
	if err != nil {
		return Page{}, err
	}

	var cursorTime time.Time
	var cursorID string
	if query.Cursor != "" {
		cursorTime, cursorID, err = DecodeCursor(query.Cursor)
		if err != nil {
			return Page{}, err
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	normalisedQuery := strings.ToLower(strings.TrimSpace(query.Text))
	status := query.Status
	matched := make([]Event, 0, len(s.events))

	for _, event := range s.events {
		if status != "" && status != "all" && string(event.Status) != status {
			continue
		}
		if query.ConnectionID != "" && event.ConnectionID != query.ConnectionID {
			continue
		}
		if normalisedQuery != "" && !matches(event, normalisedQuery) {
			continue
		}
		matched = append(matched, event)
	}

	paged := make([]Event, 0, len(matched))
	for _, event := range matched {
		if query.Cursor != "" && !afterCursor(event, cursorTime, cursorID) {
			continue
		}
		paged = append(paged, event)
	}

	next := ""
	if len(paged) > limit {
		paged = paged[:limit]
		last := paged[len(paged)-1]
		next = EncodeCursor(last.OccurredAt, last.ID)
	}

	return Page{Events: paged, Total: len(matched), NextCursor: next}, nil
}

func (s *Store) Append(_ context.Context, event Event) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.events) >= MaxEvents {
		return false, nil
	}

	s.events = append(s.events, event)
	sort.Slice(s.events, func(i, j int) bool {
		return newer(s.events[i], s.events[j])
	})
	s.notifyLocked()
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

func (s *Store) notifyLocked() {
	for ch := range s.subs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

func newer(left, right Event) bool {
	if left.OccurredAt.Equal(right.OccurredAt) {
		return left.ID > right.ID
	}
	return left.OccurredAt.After(right.OccurredAt)
}

func afterCursor(event Event, cursorTime time.Time, cursorID string) bool {
	if event.OccurredAt.Equal(cursorTime) {
		return event.ID < cursorID
	}
	return event.OccurredAt.Before(cursorTime)
}

func matches(event Event, query string) bool {
	searchable := []string{event.ID, event.Source, event.Type, event.CorrelationID}
	for _, value := range searchable {
		if strings.Contains(strings.ToLower(value), query) {
			return true
		}
	}
	return false
}

func normalizeLimit(limit int) (int, error) {
	if limit == 0 {
		return DefaultLimit, nil
	}
	if limit < 1 || limit > MaxLimit {
		return 0, errInvalidLimit
	}
	return limit, nil
}
