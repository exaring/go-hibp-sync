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

type SyncOption func(config *syncConfig)

func SyncWithContext(ctx context.Context) SyncOption {
	return func(c *syncConfig) {
		c.ctx = ctx
	}
}

func SyncWithDataDir(dataDir string) SyncOption {
	return func(c *syncConfig) {
		c.dataDir = dataDir
	}
}

func SyncWithEndpoint(endpoint string) SyncOption {
	return func(c *syncConfig) {
		c.endpoint = endpoint
	}
}

func SyncWithMinWorkers(workers int) SyncOption {
	return func(c *syncConfig) {
		c.minWorkers = workers
	}
}

func SyncWithStateFile(stateFile io.ReadWriteSeeker) SyncOption {
	return func(c *syncConfig) {
		c.stateFile = stateFile
	}
}

func SyncWithProgressFn(progressFn ProgressFunc) SyncOption {
	return func(c *syncConfig) {
		c.progressFn = progressFn
	}
}

func SyncWithNoCompression() SyncOption {
	return func(c *syncConfig) {
		c.noCompression = true
	}
}

func SyncWithLastRange(to int64) SyncOption {
	return func(c *syncConfig) {
		c.lastRange = to
	}
}

type exportConfig struct {
	commonConfig
}

type ExportOption func(*exportConfig)

func ExportWithDataDir(dataDir string) ExportOption {
	return func(c *exportConfig) {
		c.dataDir = dataDir
	}
}

func ExportWithNoCompression() ExportOption {
	return func(c *exportConfig) {
		c.noCompression = true
	}
}

type queryConfig struct {
	commonConfig
}

type QueryOption func(*queryConfig)

func QueryWithDataDir(dataDir string) QueryOption {
	return func(c *queryConfig) {
		c.dataDir = dataDir
	}
}

func QueryWithNoCompression() QueryOption {
	return func(c *queryConfig) {
		c.noCompression = true
	}
}
