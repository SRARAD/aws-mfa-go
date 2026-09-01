package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/SRARAD/aws-mfa-go/internal/app"
	"github.com/SRARAD/aws-mfa-go/internal/prompt"
)

func NewRoot(version string) *cobra.Command {
	opts := app.Options{SaveDevice: true}
	var (
		logLevel string
		doSetup  bool
	)

	cmd := &cobra.Command{
		Use:          "aws-mfa",
		Short:        "Obtain AWS STS credentials using MFA",
		SilenceUsage: true,
		Version:      version,
		Long: `aws-mfa obtains temporary AWS credentials from STS using your long-term
IAM keys plus an MFA code, then writes them to ~/.aws/credentials.

If you do not pass --device (or MFA_DEVICE / aws_mfa_device), aws-mfa lists
the MFA devices on your IAM user and lets you pick one.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			log := newLogger(logLevel, cmd.OutOrStdout())
			a := app.New(log, prompt.Default())
			if doSetup {
				return a.Setup(cmd.Context(), opts)
			}
			return a.Run(cmd.Context(), opts)
		},
	}

	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)

	cmd.Flags().StringVar(&opts.Device, "device", "", "MFA device ARN or hardware serial (also MFA_DEVICE or aws_mfa_device)")
	cmd.Flags().IntVar(&opts.Duration, "duration", 0, "temporary credential lifetime in seconds (also MFA_STS_DURATION)")
	cmd.Flags().StringVarP(&opts.Profile, "profile", "p", "", "credentials section to write the STS session into (default "+app.DefaultProfile+", also AWS_PROFILE)")
	cmd.Flags().StringVar(&opts.SourceProfile, "source-profile", "", "credentials section with long-term IAM keys (default <profile>-long-term, also MFA_SOURCE_PROFILE)")
	cmd.Flags().StringVar(&opts.SourceProfile, "from", "", "alias for --source-profile")
	cmd.Flags().StringVar(&opts.LongTermSuffix, "long-term-suffix", "", "deprecated: suffix for the long-term section (use none to omit)")
	cmd.Flags().StringVar(&opts.LongTermSuffix, "long-suffix", "", "alias for --long-term-suffix")
	cmd.Flags().StringVar(&opts.ShortTermSuffix, "short-term-suffix", "", "deprecated: suffix for the short-term section (use none to omit)")
	cmd.Flags().StringVar(&opts.ShortTermSuffix, "short-suffix", "", "alias for --short-term-suffix")
	cmd.Flags().StringVar(&opts.AssumeRole, "assume-role", "", "role ARN to assume (also MFA_ASSUME_ROLE or assume_role)")
	cmd.Flags().StringVar(&opts.AssumeRole, "assume", "", "alias for --assume-role")
	cmd.Flags().StringVar(&opts.RoleSessionName, "role-session-name", "", "session name when assuming a role (default: local username)")
	cmd.Flags().StringVarP(&opts.Region, "region", "r", "", "AWS region (default "+app.DefaultRegion+")")
	cmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "refresh credentials even if they are still valid")
	cmd.Flags().StringVarP(&opts.Token, "token", "t", "", "MFA token code (skips the interactive prompt)")
	cmd.Flags().BoolVar(&opts.SelectDevice, "select-device", false, "list MFA devices and choose one even if one is already configured")
	cmd.Flags().BoolVar(&opts.SaveDevice, "save-device", true, "write the selected MFA device to the long-term profile")
	cmd.Flags().StringVar(&opts.CredsPath, "credentials-file", "", "path to the AWS credentials file")
	cmd.Flags().BoolVar(&doSetup, "setup", false, "create a new long-term credentials section")
	cmd.PersistentFlags().StringVar(&logLevel, "log-level", "INFO", "log level: DEBUG, INFO, WARN, ERROR")

	_ = cmd.Flags().MarkHidden("long-term-suffix")
	_ = cmd.Flags().MarkHidden("long-suffix")
	_ = cmd.Flags().MarkHidden("short-term-suffix")
	_ = cmd.Flags().MarkHidden("short-suffix")
	_ = cmd.Flags().MarkHidden("assume")

	cmd.AddCommand(newSetupCmd(&opts, &logLevel))
	cmd.AddCommand(newCompletionCmd())
	return cmd
}

func newSetupCmd(opts *app.Options, logLevel *string) *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Create a new long-term credentials section",
		RunE: func(cmd *cobra.Command, _ []string) error {
			log := newLogger(*logLevel, cmd.OutOrStdout())
			return app.New(log, prompt.Default()).Setup(cmd.Context(), *opts)
		},
	}
}

func newCompletionCmd() *cobra.Command {
	return &cobra.Command{
		Use:       "completion [bash|zsh|fish|powershell]",
		Short:     "Generate shell completion scripts",
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(cmd.OutOrStdout())
			case "zsh":
				return cmd.Root().GenZshCompletion(cmd.OutOrStdout())
			case "fish":
				return cmd.Root().GenFishCompletion(cmd.OutOrStdout(), true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
			default:
				return fmt.Errorf("unsupported shell %q", args[0])
			}
		},
	}
}
