// Package scaffold is the copy source for `make new-connector NAME=...`.
// The generated package is expected to pass its test without edits.
package scaffold

import (
	"context"

	"github.com/o-mid/contract-ops/api/internal/connectors"
)

type Connector struct{}

func New() Connector { return Connector{} }

func (Connector) Kind() string { return "scaffold" }

func (Connector) Describe() connectors.Descriptor {
	return connectors.Descriptor{Kind: "scaffold", DisplayName: "scaffold"}
}

func (Connector) Verify(context.Context, connectors.Credential) error { return nil }

func (Connector) Fetch(context.Context, connectors.Credential, connectors.Window, connectors.Cursor) (connectors.Page, error) {
	return connectors.Page{Done: true}, nil
}

func (Connector) Normalize(connectors.RawRecord) ([]connectors.CostRow, error) {
	return nil, nil
}
