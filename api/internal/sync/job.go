package sync

import "time"

// streamUsage is the cursor stream CommitPage and the scheduler share.
const streamUsage = "usage"

type Job struct {
	ID           string    `json:"id"`
	ConnectionID string    `json:"connectionId"`
	WorkspaceID  string    `json:"workspaceId"`
	Kind         string    `json:"kind"`
	WindowStart  time.Time `json:"windowStart"`
	WindowEnd    time.Time `json:"windowEnd"`
	Status       string    `json:"status"`
	Attempt      int       `json:"attempt"`
	MaxAttempts  int       `json:"maxAttempts"`
	ErrorCode    string    `json:"errorCode,omitempty"`
	ErrorDetail  string    `json:"errorDetail,omitempty"`
}

type Batch struct {
	RecordCount int
	Hash        []byte
	Quarantined bool
	DriftPath   string
}
