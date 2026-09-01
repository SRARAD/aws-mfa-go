package app

import "testing"

func TestProfileNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		opts      Options
		wantLong  string
		wantShort string
		wantErr   bool
	}{
		{
			name:      "defaults",
			opts:      Options{Profile: "gdit-test"},
			wantLong:  "gdit-test-long-term",
			wantShort: "gdit-test",
		},
		{
			name:      "explicit source profile",
			opts:      Options{Profile: "myorg-production", SourceProfile: "myorg"},
			wantLong:  "myorg",
			wantShort: "myorg-production",
		},
		{
			name:    "source equal to profile rejected",
			opts:    Options{Profile: "same", SourceProfile: "same"},
			wantErr: true,
		},
		{
			name:    "empty profile rejected",
			opts:    Options{},
			wantErr: true,
		},
		{
			name: "legacy suffixes: one IAM user, several sessions",
			opts: Options{
				Profile:         "myorg",
				LongTermSuffix:  "none",
				ShortTermSuffix: "production",
			},
			wantLong:  "myorg",
			wantShort: "myorg-production",
		},
		{
			name: "legacy none is case insensitive",
			opts: Options{
				Profile:         "dev",
				LongTermSuffix:  "NONE",
				ShortTermSuffix: "None",
			},
			wantErr: true,
		},
		{
			name: "legacy explicit long suffix",
			opts: Options{
				Profile:        "acct",
				LongTermSuffix: "permanent",
			},
			wantLong:  "acct-permanent",
			wantShort: "acct",
		},
		{
			name: "legacy equal names rejected",
			opts: Options{
				Profile:         "same",
				LongTermSuffix:  "none",
				ShortTermSuffix: "none",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotLong, gotShort, err := ProfileNames(tt.opts)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got long=%q short=%q", gotLong, gotShort)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotLong != tt.wantLong || gotShort != tt.wantShort {
				t.Fatalf("got (%q, %q), want (%q, %q)", gotLong, gotShort, tt.wantLong, tt.wantShort)
			}
		})
	}
}
