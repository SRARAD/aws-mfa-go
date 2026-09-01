package app

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/SRARAD/aws-mfa/internal/awsapi"
	"github.com/SRARAD/aws-mfa/internal/creds"
	"github.com/SRARAD/aws-mfa/internal/prompt"
)

type stubDevices struct {
	devices []awsapi.Device
	err     error
}

func (s stubDevices) ListMFADevices(context.Context) ([]awsapi.Device, error) {
	return s.devices, s.err
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
}

func writeCreds(t *testing.T, body string) *creds.File {
	t.Helper()
	path := filepath.Join(t.TempDir(), "credentials")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := creds.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestResolveDeviceFlagWins(t *testing.T) {
	t.Parallel()

	file := writeCreds(t, "[p-long-term]\naws_mfa_device = arn:from-file\n")
	got, save, err := resolveDevice(context.Background(), testLogger(), Options{Device: "arn:from-flag"}, file, "p-long-term", stubDevices{}, prompt.Default())
	if err != nil {
		t.Fatal(err)
	}
	if got != "arn:from-flag" || save {
		t.Fatalf("got %q save=%v", got, save)
	}
}

func TestResolveDeviceUsesConfig(t *testing.T) {
	t.Parallel()

	file := writeCreds(t, "[p-long-term]\naws_mfa_device = arn:from-file\n")
	got, save, err := resolveDevice(context.Background(), testLogger(), Options{}, file, "p-long-term", stubDevices{err: errors.New("should not list")}, prompt.Default())
	if err != nil {
		t.Fatal(err)
	}
	if got != "arn:from-file" || save {
		t.Fatalf("got %q save=%v", got, save)
	}
}

func TestResolveDeviceSingleFromIAM(t *testing.T) {
	t.Parallel()

	file := writeCreds(t, "[p-long-term]\n")
	src := stubDevices{devices: []awsapi.Device{{SerialNumber: "arn:aws:iam::1:mfa/only"}}}
	got, save, err := resolveDevice(context.Background(), testLogger(), Options{SaveDevice: true}, file, "p-long-term", src, prompt.Default())
	if err != nil {
		t.Fatal(err)
	}
	if got != "arn:aws:iam::1:mfa/only" || !save {
		t.Fatalf("got %q save=%v", got, save)
	}
}

func TestResolveDeviceSelectsAmongMany(t *testing.T) {
	t.Parallel()

	file := writeCreds(t, "[p-long-term]\n")
	src := stubDevices{devices: []awsapi.Device{
		{SerialNumber: "arn:aws:iam::1:mfa/one"},
		{SerialNumber: "arn:aws:iam::1:mfa/two"},
	}}
	p := &prompt.IO{In: bytes.NewBufferString("2\n"), Out: &bytes.Buffer{}}
	got, save, err := resolveDevice(context.Background(), testLogger(), Options{SaveDevice: true}, file, "p-long-term", src, p)
	if err != nil {
		t.Fatal(err)
	}
	if got != "arn:aws:iam::1:mfa/two" || !save {
		t.Fatalf("got %q save=%v", got, save)
	}
}

func TestResolveDeviceListError(t *testing.T) {
	t.Parallel()

	file := writeCreds(t, "[p-long-term]\n")
	_, _, err := resolveDevice(context.Background(), testLogger(), Options{}, file, "p-long-term", stubDevices{err: errors.New("denied")}, prompt.Default())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveDeviceForceReselect(t *testing.T) {
	t.Parallel()

	file := writeCreds(t, "[p-long-term]\naws_mfa_device = arn:old\n")
	src := stubDevices{devices: []awsapi.Device{
		{SerialNumber: "arn:old"},
		{SerialNumber: "arn:new"},
	}}
	p := &prompt.IO{In: bytes.NewBufferString("2\n"), Out: &bytes.Buffer{}}
	got, save, err := resolveDevice(context.Background(), testLogger(), Options{SelectDevice: true, SaveDevice: true}, file, "p-long-term", src, p)
	if err != nil {
		t.Fatal(err)
	}
	if got != "arn:new" || !save {
		t.Fatalf("got %q save=%v", got, save)
	}
}
