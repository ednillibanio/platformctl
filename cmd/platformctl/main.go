package main

import (
	"fmt"
	"os"

	"github.com/ednillibanio/platformctl/internal/cli"
	"github.com/ednillibanio/platformctl/internal/sharedroot"
)

func main() {
	if err := cli.New().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		// Usage errors of the shared-root commands exit 2; every other
		// error, including those of the existing commands, exits 1.
		os.Exit(sharedroot.ExitCode(err))
	}
}
