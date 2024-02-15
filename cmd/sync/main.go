package main

import (
	"fmt"
	hibpsync "github.com/exaring/go-hibp-sync"
	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
	"os"
	"path"
	"time"
)

func main() {
	if err := run(); err != nil {
		_, _ = os.Stderr.WriteString("Failed to sync HIBP data: " + err.Error())

		os.Exit(1)
	}
}

func run() error {
	stateFilePath := path.Dir(hibpsync.DefaultStateFile)
	if err := os.MkdirAll(stateFilePath, 0o755); err != nil {
		return fmt.Errorf("creating state file directory %q: %w", stateFilePath, err)
	}

	stateFile, err := os.OpenFile(hibpsync.DefaultStateFile, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("opening state file: %w", err)
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

	if err := hibpsync.Sync(hibpsync.SyncWithProgressFn(updateProgressBar), hibpsync.SyncWithStateFile(stateFile)); err != nil {
		return fmt.Errorf("syncing: %w", err)
	}

	if err := os.Remove(hibpsync.DefaultStateFile); err != nil {
		return fmt.Errorf("removing state file %q: %w", hibpsync.DefaultStateFile, err)
	}

	return nil
}
