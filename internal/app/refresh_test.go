package app

import (
	"testing"
	"time"
)

func TestShouldRefresh(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	future := now.Add(30 * time.Minute)
	past := now.Add(-time.Minute)

	tests := []struct {
		name        string
		in          RefreshInput
		wantRefresh bool
		wantReason  RefreshReason
	}{
		{
			name:        "missing section",
			in:          RefreshInput{Now: now},
			wantRefresh: true,
			wantReason:  ReasonMissingSection,
		},
		{
			name: "incomplete section",
			in: RefreshInput{
				SectionExists:   true,
				SectionComplete: false,
				Now:             now,
			},
			wantRefresh: true,
			wantReason:  ReasonInvalidSection,
		},
		{
			name: "force",
			in: RefreshInput{
				Force:           true,
				SectionExists:   true,
				SectionComplete: true,
				Expiration:      future,
				Now:             now,
			},
			wantRefresh: true,
			wantReason:  ReasonForce,
		},
		{
			name: "new assume role",
			in: RefreshInput{
				SectionExists:   true,
				SectionComplete: true,
				AssumeRoleARN:   "arn:aws:iam::1:role/A",
				Expiration:      future,
				Now:             now,
			},
			wantRefresh: true,
			wantReason:  ReasonNewRole,
		},
		{
			name: "different assume role",
			in: RefreshInput{
				SectionExists:   true,
				SectionComplete: true,
				CurrentRoleARN:  "arn:aws:iam::1:role/A",
				AssumeRoleARN:   "arn:aws:iam::1:role/B",
				Expiration:      future,
				Now:             now,
			},
			wantRefresh: true,
			wantReason:  ReasonNewRole,
		},
		{
			name: "dropping assume role",
			in: RefreshInput{
				SectionExists:   true,
				SectionComplete: true,
				CurrentRoleARN:  "arn:aws:iam::1:role/A",
				Expiration:      future,
				Now:             now,
			},
			wantRefresh: true,
			wantReason:  ReasonNewRole,
		},
		{
			name: "same role still valid",
			in: RefreshInput{
				SectionExists:   true,
				SectionComplete: true,
				CurrentRoleARN:  "arn:aws:iam::1:role/A",
				AssumeRoleARN:   "arn:aws:iam::1:role/A",
				Expiration:      future,
				Now:             now,
			},
			wantRefresh: false,
			wantReason:  ReasonStillValid,
		},
		{
			name: "session token still valid",
			in: RefreshInput{
				SectionExists:   true,
				SectionComplete: true,
				Expiration:      future,
				Now:             now,
			},
			wantRefresh: false,
			wantReason:  ReasonStillValid,
		},
		{
			name: "expired",
			in: RefreshInput{
				SectionExists:   true,
				SectionComplete: true,
				Expiration:      past,
				Now:             now,
			},
			wantRefresh: true,
			wantReason:  ReasonExpired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ShouldRefresh(tt.in)
			if got.Refresh != tt.wantRefresh || got.Reason != tt.wantReason {
				t.Fatalf("got refresh=%v reason=%q, want refresh=%v reason=%q", got.Refresh, got.Reason, tt.wantRefresh, tt.wantReason)
			}
			if !tt.wantRefresh && got.Remain != 30*time.Minute {
				t.Fatalf("remain = %s, want 30m", got.Remain)
			}
		})
	}
}
