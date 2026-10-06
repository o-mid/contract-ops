// Package connectors is the vendor boundary. A connector verifies a secret,
// fetches one page, and normalizes that page into cost rows. Normalize must
// be pure: the same bytes always produce the same rows.
package connectors

import (
	"context"
	"errors"
	"time"
)

// ErrRejected means the vendor did not accept the credential.
var ErrRejected = errors.New("credential was rejected")

type Credential struct {
	Secret string
}

type Window struct {
	Start time.Time
	End   time.Time
}

type Cursor struct {
	Token string
	At    time.Time
}

type RawRecord struct {
	ID      string
	Payload []byte
}

type Page struct {
	Records []RawRecord
	Next    Cursor
	Done    bool
}

type CostRow struct {
	ProviderName      string
	BillingAccountID  string
	ServiceName       string
	ServiceCategory   string
	ResourceID        string
	SKUID             string
	ChargeCategory    string
	ChargePeriodStart time.Time
	ChargePeriodEnd   time.Time
	BilledCost        string
	EffectiveCost     string
	BillingCurrency   string
	UsageQuantity     string
	UsageUnit         string
	SourceRecordID    string
}

type Field struct {
	Name     string
	Required bool
	Secret   bool
}

type Descriptor struct {
	Kind        string
	DisplayName string
	DocsURL     string
	Fields      []Field
}

type Connector interface {
	Kind() string
	Describe() Descriptor
	Verify(ctx context.Context, cred Credential) error
	Fetch(ctx context.Context, cred Credential, window Window, cursor Cursor) (Page, error)
	Normalize(raw RawRecord) ([]CostRow, error)
}
