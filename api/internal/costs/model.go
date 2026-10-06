package costs

import "time"

type Row struct {
	ID                string
	WorkspaceID       string
	ConnectionID      string
	JobID             string
	BatchID           string
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
	IngestedAt        time.Time
}

type Page struct {
	Rows       []Row
	Total      int
	NextCursor string
}

type Query struct {
	ConnectionID string
	Start        *time.Time
	End          *time.Time
	Cursor       string
	Limit        int
}
