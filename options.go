package hibp

import (
	"context"
	"io"
)

// ProgressFunc represents a type of function that can be used to report progress of a sync operation.
// The parameters are as follows:
// - lowest: The lowest prefix that has been processed so far (due to concurrent operations, there is a window of
// prefixes that are possibly being processed at the same time, "lowest" refers to the range with the lowest prefix).
// - current: The current prefix that is being processed, i.e. for which the ProgressFunc gets invoked.
// - to: The highest prefix that will be processed.
// - processed: The number of prefixes that have been processed so far.
// - remaining: The number of prefixes that are remaining to be processed.
// The function should return an error if the operation should be aborted.
type ProgressFunc func(lowest, current, to, processed, remaining int64) error

type commonConfig struct {
	dataDir       string
	noCompression bool
}

type CommonOption func(config *commonConfig)

// WithDataDir sets the data directory for all operations.
// The directory will be created it if it does not exist.
// Default: "./.hibp-data"
func WithDataDir(dataDir string) CommonOption {
	return func(c *commonConfig) {
		c.dataDir = dataDir
	}
}

// WithNoCompression disables compression when writing/reading the file-based database.
// When the local dataset exists already, this can only be used if the dataset has been created with the same setting.
// This seriously increases the amount of storage required.
// Default: false
func WithNoCompression() CommonOption {
	return func(c *commonConfig) {
		c.noCompression = true
	}
}

type syncConfig struct {
	ctx        context.Context
	endpoint   string
	minWorkers int
	progressFn ProgressFunc
	stateFile  io.ReadWriteSeeker
	lastRange  int64
}

// SyncOption represents a type of function that can be used to customize the behavior of the Sync function.
type SyncOption func(config *syncConfig)

// SyncWithContext sets the context for the sync operation.
func SyncWithContext(ctx context.Context) SyncOption {
	return func(c *syncConfig) {
		c.ctx = ctx
	}
}

// SyncWithEndpoint sets a custom endpoint instead of the default Have-I-Been-Pwned API endpoint.
// Default: "https://api.pwnedpasswords.com/range/"
func SyncWithEndpoint(endpoint string) SyncOption {
	return func(c *syncConfig) {
		c.endpoint = endpoint
	}
}

// SyncWithMinWorkers sets the minimum number of workers goroutines that will be used to process the ranges.
// Default: 50
func SyncWithMinWorkers(workers int) SyncOption {
	return func(c *syncConfig) {
		c.minWorkers = workers
	}
}

// SyncWithStateFile sets the state file to be used for tracking progress.
// This can either be an os.File or any other implementation of io.ReadWriteSeeker.
// Seeking is only used to jump back to the start of the "virtual file".
// It should be easy enough to decorate a bytes.Buffer with the necessary methods to make it work.
// Default: nil; meaning no state will be tracked.
func SyncWithStateFile(stateFile io.ReadWriteSeeker) SyncOption {
	return func(c *syncConfig) {
		c.stateFile = stateFile
	}
}

// SyncWithProgressFn sets a custom progress function that will be called regularly.
// The function should return an error if the operation should be aborted.
// Note, there is no guarantee that the function will be called for every prefix.
// Default: no-op function
func SyncWithProgressFn(progressFn ProgressFunc) SyncOption {
	return func(c *syncConfig) {
		c.progressFn = progressFn
	}
}

// SyncWithLastRange sets the last range to be processed.
// Aside from tests, this is rarely useful.
// Default: 0xFFFFF
func SyncWithLastRange(to int64) SyncOption {
	return func(c *syncConfig) {
		c.lastRange = to
	}
}
