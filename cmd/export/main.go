package main

import (
	hibpsync "github.com/exaring/go-hibp-sync"
	"os"
)

func main() {
	if err := hibpsync.Export(os.Stdout); err != nil {
		_, _ = os.Stderr.WriteString("Failed to export HIBP data: " + err.Error())

		os.Exit(1)
	}
}
