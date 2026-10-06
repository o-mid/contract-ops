package connections

// Apply moves a connection along the health graph.
// A paused connection stays paused until something resumes it.
// needs_auth and schema_drift can be reached from healthy in one step.
func Apply(current Status, code string) Status {
	if current == StatusPaused {
		return StatusPaused
	}
	switch code {
	case "":
		return StatusHealthy
	case "auth_invalid", "auth_expired", "permission_missing":
		return StatusNeedsAuth
	case "schema_drift":
		return StatusDegraded
	case "rate_limited", "vendor_unavailable", "partial_sync":
		if current == StatusNeedsAuth {
			return StatusNeedsAuth
		}
		if current == StatusDegraded || current == StatusFailing {
			return StatusFailing
		}
		return StatusDegraded
	default:
		if current == StatusNeedsAuth {
			return StatusNeedsAuth
		}
		return StatusFailing
	}
}
