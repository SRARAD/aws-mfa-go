package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SRARAD/aws-mfa-go/internal/awsapi"
	"github.com/SRARAD/aws-mfa-go/internal/creds"
	"github.com/SRARAD/aws-mfa-go/internal/prompt"
)

type stubAPI struct {
	devices []awsapi.Device
	listErr error
	session awsapi.Session
	callErr error
	got     struct {
		duration int32
		serial   string
		token    string
		role     string
	}
}

func (s *stubAPI) ListMFADevices(context.Context) ([]awsapi.Device, error) {
	return s.devices, s.listErr
}

func (s *stubAPI) GetSessionToken(_ context.Context, duration int32, serial, token string) (awsapi.Session, error) {
	s.got.duration = duration
	s.got.serial = serial
	s.got.token = token
	return s.session, s.callErr
}

func (s *stubAPI) AssumeRole(_ context.Context, roleARN, _ string, duration int32, serial, token string) (awsapi.Session, error) {
	s.got.role = roleARN
	s.got.duration = duration
	s.got.serial = serial
	s.got.token = token
	return s.session, s.callErr
}

func newTestApp(t *testing.T, api *stubAPI, input string) *App {
	t.Helper()
	a := New(testLogger(), &prompt.IO{In: bytes.NewBufferString(input), Out: &bytes.Buffer{}})
	a.newClient = func(_, _, _ string) API { return api }
	a.now = func() time.Time { return time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC) }
	return a
}

func TestRunGetSessionToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials")
	body := `[gdit-test-long-term]
aws_access_key_id = AKIATEST
aws_secret_access_key = secret
`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	api := &stubAPI{
		devices: []awsapi.Device{{SerialNumber: "arn:aws-us-gov:iam::1:mfa/me"}},
		session: awsapi.Session{
			AccessKeyID:     "ASIA",
			SecretAccessKey: "sess",
			SessionToken:    "tok",
			Expiration:      time.Date(2026, 9, 1, 18, 0, 0, 0, time.UTC),
		},
	}
	app := newTestApp(t, api, "123456\n")
	err := app.Run(context.Background(), Options{
		Profile:    "gdit-test",
		Region:     "us-gov-west-1",
		CredsPath:  path,
		SaveDevice: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if api.got.serial != "arn:aws-us-gov:iam::1:mfa/me" || api.got.token != "123456" || api.got.duration != DefaultSessionDuration {
		t.Fatalf("unexpected STS call: %+v", api.got)
	}

	file, err := creds.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !file.ShortTermComplete("gdit-test") {
		t.Fatal("short-term section incomplete")
	}
	if file.GetOptional("gdit-test-long-term", creds.KeyMFADevice) != "arn:aws-us-gov:iam::1:mfa/me" {
		t.Fatal("expected selected device to be saved")
	}
}

func TestRunSkipsWhenValid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials")
	body := `[gdit-test-long-term]
aws_access_key_id = AKIATEST
aws_secret_access_key = secret
aws_mfa_device = arn:x

[gdit-test]
assumed_role = False
aws_access_key_id = ASIA
aws_secret_access_key = sess
aws_session_token = tok
aws_security_token = tok
expiration = 2026-09-01 18:00:00
`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	api := &stubAPI{}
	app := newTestApp(t, api, "")
	if err := app.Run(context.Background(), Options{Profile: "gdit-test", CredsPath: path}); err != nil {
		t.Fatal(err)
	}
	if api.got.token != "" {
		t.Fatal("STS should not have been called")
	}
}

func TestRunAssumeRole(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials")
	body := `[gdit-test-long-term]
aws_access_key_id = AKIATEST
aws_secret_access_key = secret
aws_mfa_device = arn:x
`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	api := &stubAPI{
		session: awsapi.Session{
			AccessKeyID:     "ASIA",
			SecretAccessKey: "sess",
			SessionToken:    "tok",
			Expiration:      time.Date(2026, 9, 1, 13, 0, 0, 0, time.UTC),
		},
	}
	app := newTestApp(t, api, "")
	err := app.Run(context.Background(), Options{
		Profile:         "gdit-test",
		CredsPath:       path,
		AssumeRole:      "arn:aws-us-gov:iam::1:role/Admin",
		RoleSessionName: "tester",
		Token:           "654321",
	})
	if err != nil {
		t.Fatal(err)
	}
	if api.got.role != "arn:aws-us-gov:iam::1:role/Admin" || api.got.duration != DefaultAssumeDuration {
		t.Fatalf("unexpected assume-role call: %+v", api.got)
	}
	file, err := creds.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if file.GetOptional("gdit-test", creds.KeyAssumedRoleARN) != "arn:aws-us-gov:iam::1:role/Admin" {
		t.Fatal("expected assumed role arn to be stored")
	}
}

func TestValidToken(t *testing.T) {
	t.Parallel()
	if !validToken("123456") {
		t.Fatal("expected valid")
	}
	if validToken("12345") || validToken("abcdef") || validToken("1234567") {
		t.Fatal("expected invalid")
	}
}
