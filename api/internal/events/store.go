package events

import (
	"sort"
	"strings"
	"sync"
	"time"
)

const MaxEvents = 8

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
}

type Page struct {
	Events []Event `json:"events"`
	Total  int     `json:"total"`
}

type Store struct {
	mu     sync.RWMutex
	events []Event
	subs   map[chan struct{}]struct{}
}

func NewStore(events []Event) *Store {
	copyOfEvents := append([]Event(nil), events...)
	sort.Slice(copyOfEvents, func(i, j int) bool {
		return copyOfEvents[i].OccurredAt.After(copyOfEvents[j].OccurredAt)
	})

	return &Store{
		events: copyOfEvents,
		subs:   make(map[chan struct{}]struct{}),
	}
}

func (s *Store) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.events)
}

func (s *Store) List(query string, status string) Page {
	s.mu.RLock()
	defer s.mu.RUnlock()

	normalisedQuery := strings.ToLower(strings.TrimSpace(query))
	filtered := make([]Event, 0, len(s.events))

	for _, event := range s.events {
		if status != "" && status != "all" && string(event.Status) != status {
			continue
		}

		if normalisedQuery != "" && !matches(event, normalisedQuery) {
			continue
		}

		filtered = append(filtered, event)
	}

	return Page{Events: filtered, Total: len(filtered)}
}

func (s *Store) Append(event Event) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.events) >= MaxEvents {
		return false
	}

	s.events = append(s.events, event)
	sort.Slice(s.events, func(i, j int) bool {
		return s.events[i].OccurredAt.After(s.events[j].OccurredAt)
	})
	s.notifyLocked()
	return true
}

func (s *Store) Subscribe() (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)

	s.mu.Lock()
	s.subs[ch] = struct{}{}
	s.mu.Unlock()

	cancel := func() {
		s.mu.Lock()
		if _, ok := s.subs[ch]; ok {
			delete(s.subs, ch)
			close(ch)
		}
		s.mu.Unlock()
	}

	return ch, cancel
}

func (s *Store) notifyLocked() {
	for ch := range s.subs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
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
