package app

import (
	"errors"
	"strings"
)

const defaultLongTermSuffix = "long-term"

// ProfileNames resolves the credentials sections to read (long-term IAM keys)
// and write (STS session).
//
// Default: --profile gdit-test → read [gdit-test-long-term], write [gdit-test].
// Override the source with --source-profile (or --from).
//
// Hidden --long-term-suffix / --short-term-suffix keep the original Python
// aws-mfa stem+suffix behavior when either is set.
func ProfileNames(opts Options) (longTerm, shortTerm string, err error) {
	profile := strings.TrimSpace(opts.Profile)
	if profile == "" {
		return "", "", errors.New("--profile is required")
	}

	if opts.LongTermSuffix != "" || opts.ShortTermSuffix != "" {
		return suffixProfileNames(profile, opts.LongTermSuffix, opts.ShortTermSuffix)
	}

	shortTerm = profile
	longTerm = strings.TrimSpace(opts.SourceProfile)
	if longTerm == "" {
		longTerm = profile + "-" + defaultLongTermSuffix
	}
	if longTerm == shortTerm {
		return "", "", errors.New("--source-profile and --profile must name different credentials sections")
	}
	return longTerm, shortTerm, nil
}

func suffixProfileNames(profile, longSuffix, shortSuffix string) (longTerm, shortTerm string, err error) {
	longTerm = applySuffix(profile, longSuffix, defaultLongTermSuffix)
	shortTerm = applySuffix(profile, shortSuffix, "")
	if longTerm == shortTerm {
		return "", "", errors.New("the value for --long-term-suffix cannot be equal to the value for --short-term-suffix")
	}
	return longTerm, shortTerm, nil
}

func applySuffix(profile, suffix, implicit string) string {
	s := strings.TrimSpace(suffix)
	if s == "" {
		if implicit == "" {
			return profile
		}
		return profile + "-" + implicit
	}
	if strings.EqualFold(s, "none") {
		return profile
	}
	return profile + "-" + s
}
