// Package main contains a small utility to sync the HIBP data to the default data directory or to the directory specified as the
// first argument.
// The data will be stored applying zstd compression.
// The tool keeps track of progress and is able to continue from where it left off in case syncing
// needs to be interrupted.
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
	dataDir := hibpsync.DefaultDataDir

	if len(os.Args) == 2 {
		dataDir = os.Args[1]
	}

	if err := run(dataDir); err != nil {
		_, _ = os.Stderr.WriteString("Failed to sync HIBP data: " + err.Error())

		os.Exit(1)
	}
}

func run(dataDir string) error {
	stateFilePath := path.Join(dataDir, hibpsync.DefaultStateFileName)
	if err := os.MkdirAll(path.Dir(stateFilePath), 0o755); err != nil {
		return fmt.Errorf("creating state file directory %q: %w", stateFilePath, err)
	}

	stateFile, err := os.OpenFile(stateFilePath, os.O_RDWR|os.O_CREATE, 0644)
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

	if err := hibpsync.Sync(
		hibpsync.SyncWithDataDir(dataDir),
		hibpsync.SyncWithProgressFn(updateProgressBar),
		hibpsync.SyncWithStateFile(stateFile)); err != nil {
		return fmt.Errorf("syncing: %w", err)
	}

	// Explicitly close the file because otherwise we cannot remove it in the next step
	stateFile.Close()

	if err := os.Remove(stateFilePath); err != nil {
		return fmt.Errorf("removing state file %q: %w", stateFilePath, err)
	}

	return nil
}
