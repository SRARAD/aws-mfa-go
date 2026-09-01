package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/SRARAD/aws-mfa/internal/creds"
)

func (a *App) Setup(ctx context.Context, opts Options) error {
	opts = ResolveDefaults(opts)
	file, err := a.loadOrCreate(opts)
	if err != nil {
		return err
	}

	profile, err := a.prompter.Line("Profile name [%s]: ", DefaultProfile)
	if err != nil {
		return err
	}
	profile = strings.TrimSpace(profile)
	if profile == "" {
		profile = DefaultProfile
	}

	longTerm := profile + "-" + defaultLongTermSuffix
	if file.HasSection(longTerm) {
		return fmt.Errorf("section [%s] already exists in %s", longTerm, file.Path())
	}

	accessKey, err := a.prompter.Secret("aws_access_key_id")
	if err != nil {
		return err
	}
	if strings.TrimSpace(accessKey) == "" {
		return errors.New("you must supply aws_access_key_id")
	}
	secretKey, err := a.prompter.Secret("aws_secret_access_key")
	if err != nil {
		return err
	}
	if strings.TrimSpace(secretKey) == "" {
		return errors.New("you must supply aws_secret_access_key")
	}

	file.Set(longTerm, creds.KeyAccessKeyID, strings.TrimSpace(accessKey))
	file.Set(longTerm, creds.KeySecretAccessKey, strings.TrimSpace(secretKey))

	client := a.newClient(strings.TrimSpace(accessKey), strings.TrimSpace(secretKey), opts.Region)
	devices, listErr := client.ListMFADevices(ctx)
	switch {
	case listErr != nil:
		a.log.Warn("Could not list MFA devices; you can set aws_mfa_device later", "error", listErr)
	case len(devices) == 1:
		file.Set(longTerm, creds.KeyMFADevice, devices[0].SerialNumber)
		a.log.Info("Saved MFA device", "device", devices[0].SerialNumber)
	case len(devices) > 1:
		labels := make([]string, len(devices))
		for i, d := range devices {
			labels[i] = d.Label()
		}
		idx, selErr := a.prompter.Select("Multiple MFA devices found. Select one to save:", labels)
		if selErr != nil {
			return selErr
		}
		file.Set(longTerm, creds.KeyMFADevice, devices[idx].SerialNumber)
		a.log.Info("Saved MFA device", "device", devices[idx].SerialNumber)
	}

	if err := file.Save(); err != nil {
		return err
	}
	a.log.Info("Wrote long-term credentials", "section", longTerm, "path", file.Path())
	return nil
}

func (a *App) loadOrCreate(opts Options) (*creds.File, error) {
	path := opts.CredsPath
	if path == "" {
		var err error
		path, err = creds.DefaultPath()
		if err != nil {
			return nil, err
		}
	}
	if file, err := a.load(path); err == nil {
		return file, nil
	}
	return a.openCredentials(path)
}
