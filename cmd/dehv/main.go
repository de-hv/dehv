// dehv is my own desktop toolkit: `dehv` opens the hub, `dehv <command>`
// runs one of its commands.
package main

import (
	"context"
	"os"

	"github.com/de-hv/dehv/internal/cli"
	"github.com/de-hv/dehv/internal/core"
	"github.com/de-hv/dehv/internal/tui"
)

// version is set by the release build: -ldflags "-X main.version=1.0.0".
var version = "dev"

func main() {
	core.Version = version
	cli.RunHub = tui.Run
	os.Exit(cli.ExitCode(cli.Execute(context.Background(), os.Args[1:])))
}
