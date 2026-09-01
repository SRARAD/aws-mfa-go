package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"strings"
	"time"

	"github.com/SRARAD/aws-mfa-go/internal/awsapi"
	"github.com/SRARAD/aws-mfa-go/internal/creds"
	"github.com/SRARAD/aws-mfa-go/internal/prompt"
)

// API is the AWS surface used by App.
type API interface {
	DeviceSource
	GetSessionToken(ctx context.Context, duration int32, serial, token string) (awsapi.Session, error)
	AssumeRole(ctx context.Context, roleARN, sessionName string, duration int32, serial, token string) (awsapi.Session, error)
}

// App is the aws-mfa program.
type App struct {
	log       *slog.Logger
	prompter  Prompter
	load      func(path string) (*creds.File, error)
	newClient func(accessKeyID, secretAccessKey, region string) API
	now       func() time.Time
}

func New(log *slog.Logger, p Prompter) *App {
	if log == nil {
		log = slog.Default()
	}
	if p == nil {
		p = prompt.Default()
	}
	return &App{
		log:      log,
		prompter: p,
		load:     creds.Load,
		newClient: func(id, secret, region string) API {
			return awsapi.New(id, secret, region)
		},
		now: func() time.Time { return time.Now().UTC() },
	}
}

func (a *App) Run(ctx context.Context, opts Options) error {
	opts = ResolveDefaults(opts)
	if opts.Duration < 0 {
		return errors.New("duration must be positive")
	}

	path := opts.CredsPath
	if path == "" {
		var err error
		path, err = creds.DefaultPath()
		if err != nil {
			return err
		}
	}

	file, err := a.openCredentials(path)
	if err != nil {
		return err
	}

	longTerm, shortTerm, err := ProfileNames(opts)
	if err != nil {
		return err
	}

	if opts.AssumeRole == "" {
		opts.AssumeRole = file.GetOptional(longTerm, creds.KeyAssumeRole)
	}
	if opts.Duration == 0 {
		opts.Duration = defaultDuration(opts.AssumeRole)
	}

	roleMsg := ""
	if opts.AssumeRole != "" {
		roleMsg = " with assumed role: " + opts.AssumeRole
	}
	a.log.Info("Validating credentials for profile: " + shortTerm + roleMsg)

	accessKeyID, secretAccessKey, err := file.LongTermKeys(longTerm)
	if err != nil {
		return err
	}

	client := a.newClient(accessKeyID, secretAccessKey, opts.Region)

	device, saveDevice, err := resolveDevice(ctx, a.log, opts, file, longTerm, client, a.prompter)
	if err != nil {
		return err
	}
	opts.Device = device

	decision := a.refreshDecision(file, shortTerm, opts)
	a.log.Info(string(decision.Reason) + remainSuffix(decision, file, shortTerm))
	if !decision.Refresh {
		return nil
	}

	token := strings.TrimSpace(opts.Token)
	if token == "" {
		token, err = a.prompter.Line(
			"Enter AWS MFA code for device [%s] (renewing for %d seconds): ",
			opts.Device,
			opts.Duration,
		)
		if err != nil {
			return err
		}
		token = strings.TrimSpace(token)
	}
	if token == "" {
		return errors.New("MFA token is required")
	}
	if !validToken(token) {
		return errors.New("token must be six digits")
	}

	duration, err := durationSeconds(opts.Duration)
	if err != nil {
		return err
	}

	var session awsapi.Session
	if opts.AssumeRole != "" {
		if opts.RoleSessionName == "" {
			return errors.New("you must specify a role session name via --role-session-name")
		}
		a.log.Info("Assuming role", "profile", shortTerm, "role", opts.AssumeRole, "duration", opts.Duration)
		session, err = client.AssumeRole(ctx, opts.AssumeRole, opts.RoleSessionName, duration, opts.Device, token)
	} else {
		a.log.Info("Fetching credentials", "profile", shortTerm, "duration", opts.Duration)
		session, err = client.GetSessionToken(ctx, duration, opts.Device, token)
	}
	if err != nil {
		return err
	}

	file.WriteSession(shortTerm, session.AccessKeyID, session.SecretAccessKey, session.SessionToken, session.Expiration, opts.AssumeRole)
	if saveDevice {
		file.Set(longTerm, creds.KeyMFADevice, opts.Device)
		a.log.Info("Saved MFA device to long-term profile", "device", opts.Device, "section", longTerm)
	}
	if err := file.Save(); err != nil {
		return err
	}
	a.log.Info(fmt.Sprintf("Success! Your credentials will expire in %d seconds at: %s", opts.Duration, session.Expiration.Format(time.RFC3339)))
	return nil
}

func (a *App) openCredentials(path string) (*creds.File, error) {
	if _, err := os.Stat(path); err == nil {
		return a.load(path)
	}
	ok, confirmErr := a.prompter.Confirm(fmt.Sprintf("Could not locate credentials file at %s, would you like to create one?", path))
	if confirmErr != nil {
		return nil, confirmErr
	}
	if !ok {
		return nil, fmt.Errorf("could not locate credentials file at %s", path)
	}
	if err := creds.CreateEmpty(path); err != nil {
		return nil, err
	}
	return a.load(path)
}

func (a *App) refreshDecision(file *creds.File, shortTerm string, opts Options) RefreshDecision {
	in := RefreshInput{
		Force:           opts.Force,
		SectionExists:   file.HasSection(shortTerm),
		SectionComplete: file.ShortTermComplete(shortTerm),
		AssumeRoleARN:   opts.AssumeRole,
		Now:             a.now(),
	}
	if in.SectionExists {
		in.CurrentRoleARN = file.GetOptional(shortTerm, creds.KeyAssumedRoleARN)
		if exp, err := file.Expiration(shortTerm); err == nil {
			in.Expiration = exp
		} else if in.SectionComplete {
			in.SectionComplete = false
		}
	}
	return ShouldRefresh(in)
}

func remainSuffix(d RefreshDecision, file *creds.File, shortTerm string) string {
	if d.Refresh || d.Reason != ReasonStillValid {
		return ""
	}
	exp, err := file.Expiration(shortTerm)
	if err != nil {
		return ""
	}
	return fmt.Sprintf(" They will expire at %s (%.0f seconds remaining).", exp.Format(creds.ExpirationLayout), d.Remain.Seconds())
}

func durationSeconds(d int) (int32, error) {
	if d < 0 || d > math.MaxInt32 {
		return 0, errors.New("duration must be a positive 32-bit integer")
	}
	return int32(d), nil
}

func validToken(token string) bool {
	if len(token) != 6 {
		return false
	}
	for _, r := range token {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
