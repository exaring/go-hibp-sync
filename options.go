package hibpsync

import (
	"context"
	"io"
)

type commonConfig struct {
	dataDir       string
	noCompression bool
}

type syncConfig struct {
	commonConfig
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

// SyncWithDataDir sets the data directory for the sync operation.
// The directory will be created it if it does not exist.
// Default: "./.hibp-data"
func SyncWithDataDir(dataDir string) SyncOption {
	return func(c *syncConfig) {
		c.dataDir = dataDir
	}
}

// SyncWithEndpoint sets a custom endpoint instead of the default HIBP API endpoint.
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
// Default: nil, i.e., no state will be tracked.
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

// SyncWithNoCompression disables compression for the sync operation.
// This seriously increases the amount of storage required.
// Default: false
func SyncWithNoCompression() SyncOption {
	return func(c *syncConfig) {
		c.noCompression = true
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

type exportConfig struct {
	commonConfig
}

// ExportOption represents a type of function that can be used to customize the behavior of the Export function.
type ExportOption func(*exportConfig)

// ExportWithDataDir sets the data directory for the export operation.
// Default: "./.hibp-data"
func ExportWithDataDir(dataDir string) ExportOption {
	return func(c *exportConfig) {
		c.dataDir = dataDir
	}
}

// ExportWithNoCompression instructs the export operation to assume the local data is not compressed.
// This should be in sync with the configuration of the call to Sync.
// Default: false
func ExportWithNoCompression() ExportOption {
	return func(c *exportConfig) {
		c.noCompression = true
	}
}

type queryConfig struct {
	commonConfig
}

// RangeAPIOption represents a type of function that can be used to customize the behavior of the RangeAPI constructor.
type RangeAPIOption func(*queryConfig)

// QueryWithDataDir sets the data directory for the RangeAPI.
// Default: "./.hibp-data"
func QueryWithDataDir(dataDir string) RangeAPIOption {
	return func(c *queryConfig) {
		c.dataDir = dataDir
	}
}

// QueryWithNoCompression instructs the RangeAPI to assume the local data is not compressed.
// This should be in sync with the configuration of the call to Sync.
// Default: false
func QueryWithNoCompression() RangeAPIOption {
	return func(c *queryConfig) {
		c.noCompression = true
	}
}
