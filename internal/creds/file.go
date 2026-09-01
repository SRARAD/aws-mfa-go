package creds

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/ini.v1"
)

const (
	KeyAccessKeyID     = "aws_access_key_id"
	KeySecretAccessKey = "aws_secret_access_key" //nolint:gosec // AWS credentials key name, not a secret
	KeySessionToken    = "aws_session_token"
	KeySecurityToken   = "aws_security_token"
	KeyExpiration      = "expiration"
	KeyAssumedRole     = "assumed_role"
	KeyAssumedRoleARN  = "assumed_role_arn"
	KeyMFADevice       = "aws_mfa_device"
	KeyAssumeRole      = "assume_role"

	ExpirationLayout = "2006-01-02 15:04:05"
)

var shortTermRequired = []string{
	KeyAssumedRole,
	KeyAccessKeyID,
	KeySecretAccessKey,
	KeySessionToken,
	KeySecurityToken,
	KeyExpiration,
}

// File is a parsed ~/.aws/credentials document.
type File struct {
	path string
	ini  *ini.File
}

// DefaultPath returns $HOME/.aws/credentials.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".aws", "credentials"), nil
}

// Load reads an AWS credentials file. The file must already exist.
func Load(path string) (*File, error) {
	cfg, err := ini.LoadSources(ini.LoadOptions{
		Insensitive:                 false,
		SkipUnrecognizableLines:     false,
		AllowPythonMultilineValues:  false,
		SpaceBeforeInlineComment:    true,
		UnescapeValueDoubleQuotes:   true,
		UnescapeValueCommentSymbols: false,
		PreserveSurroundedQuote:     true,
		IgnoreInlineComment:         true,
		AllowShadows:                false,
	}, path)
	if err != nil {
		return nil, fmt.Errorf("read credentials file %s: %w", path, err)
	}
	return &File{path: path, ini: cfg}, nil
}

// CreateEmpty creates an empty credentials file, including parent dirs.
func CreateEmpty(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create credentials directory: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return nil
		}
		return fmt.Errorf("create credentials file: %w", err)
	}
	return f.Close()
}

func (f *File) Path() string { return f.path }

func (f *File) HasSection(name string) bool {
	return f.ini.HasSection(name)
}

func (f *File) HasKey(section, key string) bool {
	sec, err := f.ini.GetSection(section)
	if err != nil {
		return false
	}
	return sec.HasKey(key)
}

func (f *File) Get(section, key string) (string, error) {
	sec, err := f.ini.GetSection(section)
	if err != nil {
		return "", fmt.Errorf("credentials section [%s] is missing", section)
	}
	k, err := sec.GetKey(key)
	if err != nil {
		return "", fmt.Errorf("section [%s] is missing %s", section, key)
	}
	return strings.TrimSpace(k.String()), nil
}

func (f *File) GetOptional(section, key string) string {
	v, err := f.Get(section, key)
	if err != nil {
		return ""
	}
	return v
}

func (f *File) EnsureSection(name string) {
	if !f.ini.HasSection(name) {
		_, _ = f.ini.NewSection(name)
	}
}

func (f *File) Set(section, key, value string) {
	f.EnsureSection(section)
	sec := f.ini.Section(section)
	sec.Key(key).SetValue(value)
}

func (f *File) DeleteKey(section, key string) {
	sec, err := f.ini.GetSection(section)
	if err != nil {
		return
	}
	sec.DeleteKey(key)
}

// LongTermKeys returns the long-term access key pair for a section.
func (f *File) LongTermKeys(section string) (accessKeyID, secretAccessKey string, err error) {
	if !f.HasSection(section) {
		return "", "", fmt.Errorf(
			"long term credentials section [%s] is missing; add it with your long term aws_access_key_id and aws_secret_access_key (or run aws-mfa setup)",
			section,
		)
	}
	accessKeyID, err = f.Get(section, KeyAccessKeyID)
	if err != nil {
		return "", "", err
	}
	secretAccessKey, err = f.Get(section, KeySecretAccessKey)
	if err != nil {
		return "", "", err
	}
	if accessKeyID == "" || secretAccessKey == "" {
		return "", "", fmt.Errorf("section [%s] has empty long-term credentials", section)
	}
	return accessKeyID, secretAccessKey, nil
}

// ShortTermComplete reports whether a short-term section has every required key.
func (f *File) ShortTermComplete(section string) bool {
	if !f.HasSection(section) {
		return false
	}
	for _, key := range shortTermRequired {
		if !f.HasKey(section, key) {
			return false
		}
	}
	return true
}

// Expiration parses the UTC expiration stored in a short-term section.
func (f *File) Expiration(section string) (time.Time, error) {
	raw, err := f.Get(section, KeyExpiration)
	if err != nil {
		return time.Time{}, err
	}
	exp, err := time.ParseInLocation(ExpirationLayout, raw, time.UTC)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse expiration %q: %w", raw, err)
	}
	return exp, nil
}

// WriteSession writes STS session credentials into the short-term section.
func (f *File) WriteSession(section, accessKeyID, secretAccessKey, sessionToken string, expiration time.Time, assumedRoleARN string) {
	f.Set(section, KeyAccessKeyID, accessKeyID)
	f.Set(section, KeySecretAccessKey, secretAccessKey)
	f.Set(section, KeySessionToken, sessionToken)
	f.Set(section, KeySecurityToken, sessionToken)
	f.Set(section, KeyExpiration, expiration.UTC().Format(ExpirationLayout))
	if assumedRoleARN != "" {
		f.Set(section, KeyAssumedRole, "True")
		f.Set(section, KeyAssumedRoleARN, assumedRoleARN)
	} else {
		f.Set(section, KeyAssumedRole, "False")
		f.DeleteKey(section, KeyAssumedRoleARN)
	}
}

// Save writes the credentials file back to disk with restrictive permissions.
func (f *File) Save() error {
	if err := os.MkdirAll(filepath.Dir(f.path), 0o700); err != nil {
		return fmt.Errorf("create credentials directory: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(f.path), ".credentials-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp credentials file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod credentials file: %w", err)
	}
	if _, err := f.ini.WriteTo(tmp); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write credentials file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close credentials file: %w", err)
	}
	if err := os.Rename(tmpName, f.path); err != nil {
		return fmt.Errorf("replace credentials file: %w", err)
	}
	return nil
}
