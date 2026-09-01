package main

import (
	"os"

	"github.com/SRARAD/aws-mfa-go/internal/cli"
)

// version is set at link time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	if err := cli.NewRoot(version).Execute(); err != nil {
		os.Exit(1)
	}
}
