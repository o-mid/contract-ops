package db

import (
	"context"
	"testing"
)

func TestOpenRejectsUnparseableURL(t *testing.T) {
	_, err := Open(context.Background(), "not a database url")
	if err == nil {
		t.Fatal("expected parse error")
	}
}
