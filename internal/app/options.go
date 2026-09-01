package app

import (
	"os"
	"os/user"
	"strconv"
	"strings"

	"github.com/SRARAD/aws-mfa/internal/prompt"
)

const (
	DefaultProfile = "gdit-test"
	DefaultRegion  = "us-gov-west-1"

	DefaultSessionDuration = 43200
	DefaultAssumeDuration  = 3600
)

// Options are the resolved CLI inputs for a run.
type Options struct {
	Device          string
	Duration        int
	Profile         string
	SourceProfile   string
	LongTermSuffix  string
	ShortTermSuffix string
	AssumeRole      string
	RoleSessionName string
	Region          string
	Force           bool
	Token           string
	SelectDevice    bool
	SaveDevice      bool
	CredsPath       string
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// ResolveDefaults fills profile, region, assume-role, duration, and device from env.
func ResolveDefaults(opts Options) Options {
	opts.Profile = firstNonEmpty(opts.Profile, os.Getenv("AWS_PROFILE"), DefaultProfile)
	opts.SourceProfile = firstNonEmpty(opts.SourceProfile, os.Getenv("MFA_SOURCE_PROFILE"))
	opts.Region = firstNonEmpty(opts.Region, os.Getenv("AWS_REGION"), os.Getenv("AWS_DEFAULT_REGION"), DefaultRegion)
	opts.Device = firstNonEmpty(opts.Device, os.Getenv("MFA_DEVICE"))
	opts.AssumeRole = firstNonEmpty(opts.AssumeRole, os.Getenv("MFA_ASSUME_ROLE"))
	if opts.Duration == 0 {
		if raw := os.Getenv("MFA_STS_DURATION"); raw != "" {
			if n, err := strconv.Atoi(raw); err == nil {
				opts.Duration = n
			}
		}
	}
	if opts.RoleSessionName == "" {
		if u, err := user.Current(); err == nil && u.Username != "" {
			opts.RoleSessionName = u.Username
		} else {
			opts.RoleSessionName = firstNonEmpty(os.Getenv("USER"), "aws-mfa")
		}
	}
	return opts
}

func defaultDuration(assumeRole string) int {
	if assumeRole != "" {
		return DefaultAssumeDuration
	}
	return DefaultSessionDuration
}

// Prompter is the interactive surface used by App.
type Prompter interface {
	Confirm(question string) (bool, error)
	Line(format string, args ...any) (string, error)
	Secret(label string) (string, error)
	Select(title string, labels []string) (int, error)
}

var _ Prompter = (*prompt.IO)(nil)
