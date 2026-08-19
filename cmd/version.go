package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version and GitCommit are injected at build time via -ldflags -X, targeting
// this (cmd) package — see the Makefile.
var (
	Version   string
	GitCommit string
)

// MakeVersion returns the version subcommand.
func MakeVersion() *cobra.Command {
	var command = &cobra.Command{
		Use:   "version",
		Short: "Print vzn version and build metadata",
	}
	command.Run = func(cmd *cobra.Command, args []string) {
		if len(Version) == 0 {
			fmt.Println("Version: dev")
		} else {
			fmt.Println("Version:", Version)
		}
		fmt.Println("Git Commit:", GitCommit)
	}
	return command
}
