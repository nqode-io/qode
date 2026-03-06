package main

import (
	"fmt"
	"os"

	"github.com/nqode/qode/internal/cli"
	"github.com/nqode/qode/internal/config"
	"github.com/nqode/qode/internal/env"
)

// version is set at build time via ldflags.
var version = "dev"

func main() {
	cli.SetVersion(version)
	loadDotEnv()
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

// loadDotEnv loads .env from the project root before any command runs.
// Errors are non-fatal: a warning is printed and execution continues.
func loadDotEnv() {
	root, err := config.FindRoot(".")
	if err != nil {
		wd, wdErr := os.Getwd()
		if wdErr != nil {
			return
		}
		root = wd
	}

	if err := env.Load(root); err != nil {
		fmt.Fprintln(os.Stderr, "Warning: could not load .env file:", err)
	}
}
