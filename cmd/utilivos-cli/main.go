package main

import (
	"fmt"
	"os"

	"github.com/utilivos/utilivos-cli/internal/cli"
)

func main() {
	code, err := cli.Run(os.Args, os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	os.Exit(code)
}
