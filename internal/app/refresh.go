package app

import "time"

// RefreshReason explains why credentials will or will not be refreshed.
type RefreshReason string

const (
	ReasonForce          RefreshReason = "Forcing refresh of credentials."
	ReasonMissingSection RefreshReason = "Short term credentials section is missing, obtaining new credentials."
	ReasonInvalidSection RefreshReason = "Your existing credentials are missing or invalid, obtaining new credentials."
	ReasonNewRole        RefreshReason = "Obtaining credentials for a new role or profile."
	ReasonExpired        RefreshReason = "Your credentials have expired, renewing."
	ReasonStillValid     RefreshReason = "Your credentials are still valid."
)

// RefreshInput is the data needed to decide whether to call STS.
type RefreshInput struct {
	Force           bool
	SectionExists   bool
	SectionComplete bool
	CurrentRoleARN  string
	AssumeRoleARN   string
	Expiration      time.Time
	Now             time.Time
}

// RefreshDecision is the result of ShouldRefresh.
type RefreshDecision struct {
	Refresh bool
	Reason  RefreshReason
	Remain  time.Duration
}

// ShouldRefresh decides whether short-term credentials need to be renewed.
func ShouldRefresh(in RefreshInput) RefreshDecision {
	if !in.SectionExists {
		return RefreshDecision{Refresh: true, Reason: ReasonMissingSection}
	}
	if !in.SectionComplete {
		return RefreshDecision{Refresh: true, Reason: ReasonInvalidSection}
	}
	if in.Force {
		return RefreshDecision{Refresh: true, Reason: ReasonForce}
	}

	current := in.CurrentRoleARN
	wanted := in.AssumeRoleARN
	switch {
	case current == "" && wanted != "":
		return RefreshDecision{Refresh: true, Reason: ReasonNewRole}
	case current != "" && wanted != "" && current != wanted:
		return RefreshDecision{Refresh: true, Reason: ReasonNewRole}
	case current != "" && wanted == "":
		return RefreshDecision{Refresh: true, Reason: ReasonNewRole}
	}

	remain := in.Expiration.Sub(in.Now)
	if remain <= 0 {
		return RefreshDecision{Refresh: true, Reason: ReasonExpired}
	}
	return RefreshDecision{Refresh: false, Reason: ReasonStillValid, Remain: remain}
}
