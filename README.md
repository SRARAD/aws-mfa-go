# aws-mfa

<p align="center">
  <a href="https://github.com/SRARAD/aws-mfa/actions/workflows/ci.yml"><img src="https://github.com/SRARAD/aws-mfa/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/SRARAD/aws-mfa/actions/workflows/release.yml"><img src="https://github.com/SRARAD/aws-mfa/actions/workflows/release.yml/badge.svg" alt="Release"></a>
  <a href="https://github.com/SRARAD/aws-mfa/actions/workflows/ci.yml"><img src="https://github.com/SRARAD/aws-mfa/raw/gh-pages/badges/coverage.svg" alt="Coverage"></a>
  <a href="https://pkg.go.dev/github.com/SRARAD/aws-mfa"><img src="https://pkg.go.dev/badge/github.com/SRARAD/aws-mfa.svg" alt="Go Reference"></a>
  <img src="https://img.shields.io/github/go-mod/go-version/SRARAD/aws-mfa" alt="Go version">
  <a href="LICENSE"><img src="https://img.shields.io/github/license/SRARAD/aws-mfa" alt="License"></a>
</p>

A Go CLI that obtains temporary AWS credentials from STS using your long-term IAM keys and an MFA code, then writes them to `~/.aws/credentials`.

This is a Go rewrite of the original Python `aws-mfa` tool. In addition to the original flags, it can **list your IAM MFA devices and let you pick one** so you no longer have to copy a device ARN by hand.

## Why two credential sections?

* **long-term** — your IAM access key and secret, stored as `[<profile>-long-term]` (override with `--source-profile`)
* **short-term** — STS session credentials that the AWS SDKs actually use, stored as `[<profile>]` (`--profile` / `AWS_PROFILE`)

`aws-mfa` reads the long-term keys, prompts for an MFA code (or lists devices first), calls STS, and updates the short-term section.

## Install

### Binary (no Go)

Browse [GitHub Releases](https://github.com/SRARAD/aws-mfa/releases/latest) and download the archive for your platform:

| Platform | Archive |
| --- | --- |
| Linux x86_64 | `aws-mfa_<version>_linux_amd64.tar.gz` |
| Linux ARM64 | `aws-mfa_<version>_linux_arm64.tar.gz` |
| macOS Apple Silicon | `aws-mfa_<version>_darwin_arm64.tar.gz` |
| macOS Intel | `aws-mfa_<version>_darwin_amd64.tar.gz` |
| Windows x86_64 | `aws-mfa_<version>_windows_amd64.zip` |

Unpack and put `aws-mfa` on your `PATH` (for example `~/.local/bin`):

```sh
tar xzf aws-mfa_*_linux_amd64.tar.gz
chmod +x aws-mfa
mv aws-mfa ~/.local/bin/
aws-mfa --version
```

Or install the latest GitHub-hosted binary with curl (Linux / macOS):

```sh
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
esac
REPO=SRARAD/aws-mfa
TAG=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | head -1 | cut -d '"' -f4)
VER=${TAG#v}
mkdir -p "$HOME/.local/bin"
curl -fsSL "https://github.com/${REPO}/releases/download/${TAG}/aws-mfa_${VER}_${OS}_${ARCH}.tar.gz" \
  | tar -xz -C "$HOME/.local/bin" aws-mfa
```

Pin a version by replacing `TAG` / `VER` (for example `TAG=v0.1.0` and `VER=0.1.0`).

If you use the GitHub CLI:

```sh
gh release download -R SRARAD/aws-mfa --pattern "aws-mfa_*_linux_amd64.tar.gz"
tar xzf aws-mfa_*_linux_amd64.tar.gz
mv aws-mfa ~/.local/bin/
```

### With Go

Requires Go 1.26 or later ([install Go](https://go.dev/doc/install)). `$GOPATH/bin` (usually `~/go/bin`) should be on your `PATH`.

```sh
go install github.com/SRARAD/aws-mfa/cmd/aws-mfa@latest
aws-mfa --version
```

### From source

```sh
git clone https://github.com/SRARAD/aws-mfa.git
cd aws-mfa
make install
```

Or without installing into `GOBIN`:

```sh
make build
./aws-mfa --help
```

## Credentials file

Long-term keys live in a `-long-term` section. After a successful run, the short-term profile is filled in:

```ini
[gdit-test-long-term]
aws_access_key_id = YOUR_LONGTERM_KEY_ID
aws_secret_access_key = YOUR_LONGTERM_ACCESS_KEY
aws_mfa_device = arn:aws-us-gov:iam::123456789012:mfa/you

[gdit-test]
aws_access_key_id = <POPULATED_BY_AWS-MFA>
aws_secret_access_key = <POPULATED_BY_AWS-MFA>
aws_session_token = <POPULATED_BY_AWS-MFA>
aws_security_token = <POPULATED_BY_AWS-MFA>
expiration = 2026-09-01 18:00:00
assumed_role = False
```

`aws_mfa_device` is optional. If it is missing, `aws-mfa` calls `iam:ListMFADevices` with your long-term keys, then:

* uses the only device if there is just one
* prompts you to choose if there are several
* saves the choice back to `aws_mfa_device` (disable with `--save-device=false`)

Interactive setup:

```sh
aws-mfa setup
```

That writes the long-term section and tries to discover/save an MFA device the same way.

## Usage

Defaults for this fork: profile `gdit-test`, region `us-gov-west-1`.

```sh
# List MFA devices if none is configured, prompt for a code, write short-term creds
aws-mfa

# Force a new session even if the current one is still valid
aws-mfa --force

# Re-pick an MFA device (even if aws_mfa_device is already set)
aws-mfa --select-device

# Non-interactive: pass the device and token yourself
aws-mfa --device arn:aws-us-gov:iam::123456789012:mfa/you --token 123456

# Assume a role
aws-mfa --assume-role arn:aws-us-gov:iam::123456789012:role/Admin --role-session-name my-session
```

### Flags

| Flag | Also | Meaning |
| --- | --- | --- |
| `--device` | `MFA_DEVICE`, `aws_mfa_device` | MFA serial / ARN. If omitted, devices are listed from IAM. |
| `--duration` | `MFA_STS_DURATION` | Session lifetime in seconds. Default `43200`, or `3600` with `--assume-role`. |
| `--profile`, `-p` | `AWS_PROFILE` | Credentials section to **write** the STS session into. Default `gdit-test`. |
| `--source-profile`, `--from` | `MFA_SOURCE_PROFILE` | Credentials section to **read** long-term IAM keys from. Default `<profile>-long-term`. |
| `--region`, `-r` | `AWS_REGION` | AWS region. Default `us-gov-west-1`. |
| `--token`, `-t` | | MFA code. Skips the prompt. |
| `--force`, `-f` | | Refresh even if credentials are still valid. |
| `--select-device` | | Always list/select an MFA device. |
| `--save-device` | | Persist the selected device (default true). |
| `--assume-role`, `--assume` | `MFA_ASSUME_ROLE`, `assume_role` | Role ARN to assume. |
| `--role-session-name` | | Session name for AssumeRole (default: local username). |
| `--credentials-file` | | Alternate credentials path. |
| `--setup` | | Same as `aws-mfa setup`. |
| `--log-level` | | `DEBUG`, `INFO`, `WARN`, `ERROR`. Default `INFO`. |

Command-line flags override environment variables.

Shell completions:

```sh
aws-mfa completion bash
aws-mfa completion zsh
```

## IAM permissions

Your long-term user needs:

* `sts:GetSessionToken` (and `sts:AssumeRole` if you use `--assume-role`)
* `iam:ListMFADevices` so the device selector can run

If listing devices is denied, pass `--device` or set `aws_mfa_device` in the long-term section.

## Multi-account sessions

One long-term IAM user, several session profiles. `--profile` is the section AWS tools will use; `--source-profile` is where the IAM keys live:

```sh
aws-mfa --source-profile myorg --profile myorg-production --assume-role arn:aws:iam::222222222222:role/Administrator
aws-mfa --from myorg -p myorg-staging --assume-role arn:aws:iam::333333333333:role/Administrator
```

That yields `[myorg]` (keys), `[myorg-production]`, and `[myorg-staging]` in `~/.aws/credentials`.

## Development

```sh
make help                 # list targets
make tools                # install goimports-reviser + golangci-lint v2
make install-hooks        # git pre-commit: autofix + lint + test
make check                # same checks as the pre-commit hook
make test                 # go test ./...
make lint                 # golangci-lint (no write)
make coverage-badge       # coverage.out + badges/coverage.svg
make build                # compile ./aws-mfa
```

CI on every PR and push to `main` builds, runs golangci-lint, and tests. Pushes to `main` also write `badges/coverage.svg` to the `gh-pages` branch (the coverage badge in this README).

## Releasing

The current release is tracked in [`VERSION`](VERSION) and as a git tag. Pushing a `v*` tag runs **GoReleaser** and publishes multi-platform CLI binaries to [GitHub Releases](https://github.com/SRARAD/aws-mfa/releases).

```sh
make version                 # show VERSION file + latest tag
make release                 # patch bump (v0.1.0 → v0.1.1), commit VERSION, tag, push
make release BUMP=minor      # v0.1.0 → v0.2.0
make release BUMP=major      # v0.1.0 → v1.0.0
make release TAG=v0.2.0      # set an explicit version
make release DRY_RUN=1       # print next version only
```

Working tree must be clean. Install a released version:

```sh
go install github.com/SRARAD/aws-mfa/cmd/aws-mfa@v0.1.0
```
