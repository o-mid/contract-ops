package events

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"
)

var (
	// ErrInvalidCursor means the client sent a page token this process cannot read.
	ErrInvalidCursor = errors.New("invalid cursor")
	errInvalidLimit  = errors.New("limit must be from 1 to 100")
)

// EncodeCursor hides the keyset (occurred_at, id) from clients.
func EncodeCursor(occurredAt time.Time, id string) string {
	payload := occurredAt.UTC().Format(time.RFC3339Nano) + "\n" + id
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

// DecodeCursor reverses EncodeCursor.
func DecodeCursor(cursor string) (time.Time, string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, "", ErrInvalidCursor
	}
	occurredAt, id, ok := strings.Cut(string(raw), "\n")
	if !ok || id == "" {
		return time.Time{}, "", ErrInvalidCursor
	}
	parsed, err := time.Parse(time.RFC3339Nano, occurredAt)
	if err != nil {
		return time.Time{}, "", ErrInvalidCursor
	}
	return parsed, id, nil
}
