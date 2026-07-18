// Command ms365 is a command-line tool for Microsoft 365 via the Microsoft Graph API.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/jjuanrivvera/ms365-cli/commands"
	"github.com/jjuanrivvera/ms365-cli/internal/output"
	"github.com/jjuanrivvera/ms365-cli/internal/version"
)

func main() {
	// signal.NotifyContext makes Ctrl-C (SIGINT/SIGTERM) cancel in-flight work: pagination,
	// retry backoff, and the device-code wait all observe this context.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	root := commands.NewRootCmd()
	root.Version = version.Get().Version
	root.SetVersionTemplate(version.String() + "\n")

	// Expand user-defined aliases BEFORE cobra parses, so an alias can map to any command
	// without shadowing a built-in.
	root.SetArgs(commands.ExpandAliases(os.Args[1:]))

	if err := root.ExecuteContext(ctx); err != nil {
		// Error text can carry API-returned free text (a subject line, a display name);
		// strip terminal escapes before printing so a crafted value can't hijack the terminal.
		fmt.Fprintln(os.Stderr, "Error:", output.SanitizeTerminal(err.Error()))
		os.Exit(1)
	}
}
