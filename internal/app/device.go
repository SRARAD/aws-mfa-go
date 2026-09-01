package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/SRARAD/aws-mfa-go/internal/awsapi"
	"github.com/SRARAD/aws-mfa-go/internal/creds"
)

// DeviceSource lists MFA devices for the calling user.
type DeviceSource interface {
	ListMFADevices(ctx context.Context) ([]awsapi.Device, error)
}

func resolveDevice(ctx context.Context, log *slog.Logger, opts Options, file *creds.File, longTerm string, src DeviceSource, p Prompter) (device string, save bool, err error) {
	if opts.Device != "" && !opts.SelectDevice {
		return opts.Device, false, nil
	}

	configured := file.GetOptional(longTerm, creds.KeyMFADevice)
	if configured != "" && !opts.SelectDevice {
		return configured, false, nil
	}

	log.Info("No MFA device specified; listing devices from IAM")
	devices, err := src.ListMFADevices(ctx)
	if err != nil {
		if configured != "" {
			log.Warn("Failed to list MFA devices, using configured device", "error", err)
			return configured, false, nil
		}
		return "", false, fmt.Errorf("%w\nProvide --device, set MFA_DEVICE, or set aws_mfa_device in [%s]", err, longTerm)
	}
	if len(devices) == 0 {
		return "", false, errors.New("IAM returned no MFA devices; provide --device or enroll a device in the AWS console")
	}

	idx := 0
	if len(devices) == 1 {
		log.Info("Using the only MFA device on this account", "device", devices[0].SerialNumber)
	} else {
		labels := make([]string, len(devices))
		for i, d := range devices {
			labels[i] = d.Label()
		}
		idx, err = p.Select("Multiple MFA devices found. Select one:", labels)
		if err != nil {
			return "", false, err
		}
	}

	chosen := strings.TrimSpace(devices[idx].SerialNumber)
	save = opts.SaveDevice && chosen != configured
	return chosen, save, nil
}
