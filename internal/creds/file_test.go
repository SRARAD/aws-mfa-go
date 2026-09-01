package creds

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadRoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "credentials")
	src := `[gdit-test-long-term]
aws_access_key_id = AKIATEST
aws_secret_access_key = secret
aws_mfa_device = arn:aws-us-gov:iam::123:mfa/me
assume_role = arn:aws-us-gov:iam::123:role/Admin
`
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}

	f, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	id, secret, err := f.LongTermKeys("gdit-test-long-term")
	if err != nil {
		t.Fatal(err)
	}
	if id != "AKIATEST" || secret != "secret" {
		t.Fatalf("keys = %q %q", id, secret)
	}
	if got := f.GetOptional("gdit-test-long-term", KeyMFADevice); got != "arn:aws-us-gov:iam::123:mfa/me" {
		t.Fatalf("device = %q", got)
	}

	exp := time.Date(2026, 9, 1, 18, 0, 0, 0, time.UTC)
	f.WriteSession("gdit-test", "ASIA", "sess-secret", "token", exp, "")
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}

	reloaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reloaded.ShortTermComplete("gdit-test") {
		t.Fatal("expected complete short-term section")
	}
	gotExp, err := reloaded.Expiration("gdit-test")
	if err != nil {
		t.Fatal(err)
	}
	if !gotExp.Equal(exp) {
		t.Fatalf("expiration = %s, want %s", gotExp, exp)
	}
	if reloaded.GetOptional("gdit-test", KeyAssumedRole) != "False" {
		t.Fatal("expected assumed_role False")
	}
	if reloaded.HasKey("gdit-test", KeyAssumedRoleARN) {
		t.Fatal("did not expect assumed_role_arn")
	}
}

func TestLongTermKeysMissingSection(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "credentials")
	if err := os.WriteFile(path, []byte("[other]\naws_access_key_id = x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.LongTermKeys("missing-long-term"); err == nil {
		t.Fatal("expected error")
	}
}

func TestWriteSessionAssumedRole(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "credentials")
	if err := os.WriteFile(path, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteSession("dev", "ASIA", "s", "t", time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), "arn:aws:iam::1:role/R")
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}
	reloaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.GetOptional("dev", KeyAssumedRoleARN) != "arn:aws:iam::1:role/R" {
		t.Fatal(reloaded.GetOptional("dev", KeyAssumedRoleARN))
	}
}
