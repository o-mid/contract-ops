package events

import (
	"sort"
	"strings"
	"time"
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
}

type Page struct {
	Events []Event `json:"events"`
	Total  int     `json:"total"`
}

type Store struct {
	events []Event
}

func NewStore(events []Event) Store {
	copyOfEvents := append([]Event(nil), events...)
	sort.Slice(copyOfEvents, func(i, j int) bool {
		return copyOfEvents[i].OccurredAt.After(copyOfEvents[j].OccurredAt)
	})

	return Store{events: copyOfEvents}
}

func (s Store) List(query string, status string) Page {
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

func matches(event Event, query string) bool {
	searchable := []string{event.ID, event.Source, event.Type, event.CorrelationID}
	for _, value := range searchable {
		if strings.Contains(strings.ToLower(value), query) {
			return true
		}
	}

	return false
}
