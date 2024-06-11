// Package main contains a small utility to export the HIBP data to stdout.
// Expects the data to be available in the default data directory or in the directory specified as the first argument.
// Data is expected to be compressed.
package main

import (
	hibp "github.com/exaring/go-hibp-sync"
	"os"
)

func main() {
	dataDir := hibp.DefaultDataDir

	if len(os.Args) == 2 {
		dataDir = os.Args[1]
	}

	h, err := hibp.New(hibp.WithDataDir(dataDir))
	if err != nil {
		_, _ = os.Stderr.WriteString("Failed to init HIBP sync: " + err.Error())

		os.Exit(1)
	}

	if err := h.Export(os.Stdout); err != nil {
		_, _ = os.Stderr.WriteString("Failed to export HIBP data: " + err.Error())

		os.Exit(1)
	}
}
