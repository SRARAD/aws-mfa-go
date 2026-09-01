package awsapi

import (
	"strings"
	"testing"
	"time"
)

func TestDeviceLabel(t *testing.T) {
	t.Parallel()

	virtual := Device{SerialNumber: "arn:aws-us-gov:iam::1:mfa/me"}
	if virtual.Hardware() {
		t.Fatal("ARN should be virtual")
	}
	if !strings.Contains(virtual.Label(), "virtual") {
		t.Fatalf("label = %q", virtual.Label())
	}

	hw := Device{
		SerialNumber: "GAHT12345678",
		EnableDate:   time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
	}
	if !hw.Hardware() {
		t.Fatal("serial should be hardware")
	}
	if !strings.Contains(hw.Label(), "hardware") || !strings.Contains(hw.Label(), "2024-01-02") {
		t.Fatalf("label = %q", hw.Label())
	}
}
