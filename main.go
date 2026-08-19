// Command vzzn is a lean vision/OCR client for the toilgate gateway.
//
// Thin entrypoint: assembles and executes the cobra command tree defined in
// the cmd package. Version/build metadata is injected into the cmd package at
// build time via -ldflags -X (see Makefile), matching the k3sup/arkade layout.
package main

import (
	"fmt"
	"os"

	"github.com/alexellis/vzzn/cmd"
)

func main() {
	root := cmd.MakeRoot()
	root.SilenceUsage = true
	root.SilenceErrors = true
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "vzzn:", err)
		os.Exit(1)
	}
}
