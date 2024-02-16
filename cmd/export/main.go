package main

import (
	hibpsync "github.com/exaring/go-hibp-sync"
	"os"
)

func main() {
	dataDir := hibpsync.DefaultDataDir

	if len(os.Args) == 2 {
		dataDir = os.Args[1]
	}

	if err := hibpsync.Export(os.Stdout, hibpsync.ExportWithDataDir(dataDir)); err != nil {
		_, _ = os.Stderr.WriteString("Failed to export HIBP data: " + err.Error())

		os.Exit(1)
	}
}
