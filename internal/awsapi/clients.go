package awsapi

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	ststypes "github.com/aws/aws-sdk-go-v2/service/sts/types"
)

// Device is an IAM MFA device associated with the caller.
type Device struct {
	SerialNumber string
	EnableDate   time.Time
}

// Hardware reports whether the serial looks like a hardware token (not an ARN).
func (d Device) Hardware() bool {
	return !strings.HasPrefix(d.SerialNumber, "arn:")
}

func (d Device) Label() string {
	kind := "virtual"
	if d.Hardware() {
		kind = "hardware"
	}
	if d.EnableDate.IsZero() {
		return fmt.Sprintf("%s (%s)", d.SerialNumber, kind)
	}
	return fmt.Sprintf("%s (%s, enabled %s)", d.SerialNumber, kind, d.EnableDate.UTC().Format("2006-01-02"))
}

// Session is a set of temporary STS credentials.
type Session struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Expiration      time.Time
}

// Client talks to IAM and STS using long-term keys.
type Client struct {
	iam IAMAPI
	sts STSAPI
}

// IAMAPI is the IAM subset we use.
type IAMAPI interface {
	ListMFADevices(ctx context.Context, params *iam.ListMFADevicesInput, optFns ...func(*iam.Options)) (*iam.ListMFADevicesOutput, error)
}

// STSAPI is the STS subset we use.
type STSAPI interface {
	GetSessionToken(ctx context.Context, params *sts.GetSessionTokenInput, optFns ...func(*sts.Options)) (*sts.GetSessionTokenOutput, error)
	AssumeRole(ctx context.Context, params *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error)
}

// New builds IAM and STS clients from long-term static credentials.
func New(accessKeyID, secretAccessKey, region string) *Client {
	cfg := aws.Config{
		Region: region,
		Credentials: credentials.NewStaticCredentialsProvider(
			accessKeyID,
			secretAccessKey,
			"",
		),
	}
	return &Client{
		iam: iam.NewFromConfig(cfg),
		sts: sts.NewFromConfig(cfg),
	}
}

// NewWithAPIs is used by tests.
func NewWithAPIs(iamAPI IAMAPI, stsAPI STSAPI) *Client {
	return &Client{iam: iamAPI, sts: stsAPI}
}

// ListMFADevices returns every MFA device for the calling IAM user.
func (c *Client) ListMFADevices(ctx context.Context) ([]Device, error) {
	var (
		devices   []Device
		marker    *string
		truncated = true
	)
	for truncated {
		out, err := c.iam.ListMFADevices(ctx, &iam.ListMFADevicesInput{
			Marker: marker,
		})
		if err != nil {
			return nil, fmt.Errorf("list MFA devices: %w", err)
		}
		for _, d := range out.MFADevices {
			dev := Device{}
			if d.SerialNumber != nil {
				dev.SerialNumber = *d.SerialNumber
			}
			if d.EnableDate != nil {
				dev.EnableDate = *d.EnableDate
			}
			if dev.SerialNumber != "" {
				devices = append(devices, dev)
			}
		}
		truncated = out.IsTruncated
		marker = out.Marker
	}
	return devices, nil
}

// GetSessionToken requests temporary credentials for the calling user.
func (c *Client) GetSessionToken(ctx context.Context, duration int32, serial, token string) (Session, error) {
	out, err := c.sts.GetSessionToken(ctx, &sts.GetSessionTokenInput{
		DurationSeconds: aws.Int32(duration),
		SerialNumber:    aws.String(serial),
		TokenCode:       aws.String(token),
	})
	if err != nil {
		return Session{}, fmt.Errorf("get session token: %w", err)
	}
	return sessionFrom(out.Credentials)
}

// AssumeRole requests temporary credentials for a role, MFA-authenticated.
func (c *Client) AssumeRole(ctx context.Context, roleARN, sessionName string, duration int32, serial, token string) (Session, error) {
	out, err := c.sts.AssumeRole(ctx, &sts.AssumeRoleInput{
		RoleArn:         aws.String(roleARN),
		RoleSessionName: aws.String(sessionName),
		DurationSeconds: aws.Int32(duration),
		SerialNumber:    aws.String(serial),
		TokenCode:       aws.String(token),
	})
	if err != nil {
		return Session{}, fmt.Errorf("assume role: %w", err)
	}
	return sessionFrom(out.Credentials)
}

func sessionFrom(c *ststypes.Credentials) (Session, error) {
	if c == nil || c.AccessKeyId == nil || c.SecretAccessKey == nil || c.SessionToken == nil || c.Expiration == nil {
		return Session{}, errors.New("STS returned incomplete credentials")
	}
	return Session{
		AccessKeyID:     *c.AccessKeyId,
		SecretAccessKey: *c.SecretAccessKey,
		SessionToken:    *c.SessionToken,
		Expiration:      *c.Expiration,
	}, nil
}
