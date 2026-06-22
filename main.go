package main

import (
	"os"

	"github.com/huski-inc/tmcopilot-cli/cmd/tmc"
)

func main() {
	os.Exit(tmc.Execute(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
