package main

import (
	"fmt"
	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
	"os"
	"time"

	hibpsync "github.com/exaring/go-hibp-sync"
)

func main() {
	stateFile, err := os.OpenFile(hibpsync.DefaultStateFile, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		fmt.Printf("opening state file error: %q", err)
	}
	defer stateFile.Close()

	bar := progressbar.NewOptions(0xFFFFF+1,
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetDescription("[cyan]Syncing HIBP data...[reset]"),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetItsString("prefixes"),
		progressbar.OptionThrottle(100*time.Millisecond),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionSetElapsedTime(true),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))

	updateProgressBar := func(_, _, _, processed, remaining int64) error {
		_ = bar.Set64(processed)

		if remaining == 0 {
			_ = bar.Finish()
		}

		return nil
	}

	if err := hibpsync.Sync(hibpsync.WithProgressFn(updateProgressBar), hibpsync.WithStateFile(stateFile)); err != nil {
		fmt.Printf("sync error: %q", err)
	}

	if err := os.Remove(hibpsync.DefaultStateFile); err != nil {
		fmt.Printf("removing state file error: %q", err)
	}
}
