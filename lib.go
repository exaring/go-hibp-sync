package hibpsync

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/alitto/pond"
	"github.com/hashicorp/go-retryablehttp"
	"io"
	"os"
	"strconv"
)

const (
	DefaultDataDir       = "./.hibp-data"
	DefaultStateFileName = "state"
	defaultEndpoint      = "https://api.pwnedpasswords.com/range/"
	defaultWorkers       = 50
	defaultLastRange     = 0xFFFFF
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

// Sync copies the ranges, i.e., the HIBP data, from the upstream API to the local storage.
// The function will start from the lowest prefix and continue until the highest prefix.
// See the set of SyncOption functions for customizing the behavior of the sync operation.
func Sync(options ...SyncOption) error {
	config := &syncConfig{
		commonConfig: commonConfig{
			dataDir: DefaultDataDir,
		},
		ctx:        context.Background(),
		endpoint:   defaultEndpoint,
		minWorkers: defaultWorkers,
		progressFn: func(_, _, _, _, _ int64) error { return nil },
		lastRange:  defaultLastRange,
	}

	for _, option := range options {
		option(config)
	}

	from := int64(0x00000)

	if config.stateFile != nil {
		lastState, err := readStateFile(config.stateFile)
		if err != nil {
			return fmt.Errorf("error reading state file: %w", err)
		}

		from = lastState

		config.progressFn = wrapWithStateUpdate(lastState, config.stateFile, config.progressFn)
	}

	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 10
	retryClient.Logger = nil // For now, we simply want to suppress the debug output

	client := &hibpClient{
		endpoint:   config.endpoint,
		httpClient: retryClient.StandardClient(),
		maxRetries: 3,
	}

	storage := newFSStorage(config.dataDir, config.noCompression)

	// It is important to create a non-buffering/blocking pool because we don't want to schedule all jobs upfront.
	// This would cause problems, especially when cancelling the context.
	pool := pond.New(config.minWorkers, 0, pond.MinWorkers(config.minWorkers))

	return sync(config.ctx, from, config.lastRange+1, client, storage, pool, config.progressFn)
}

// Export writes the HIBP data to the given writer.
// The data is written in the same format as it is provided by the HIBP API itself.
// See the set of ExportOption functions for customizing the behavior of the export operation.
func Export(w io.Writer, options ...ExportOption) error {
	config := &exportConfig{
		commonConfig: commonConfig{
			dataDir: DefaultDataDir,
		},
	}

	for _, option := range options {
		option(config)
	}

	storage := newFSStorage(config.dataDir, config.noCompression)

	return export(0, defaultLastRange+1, storage, w)
}

// RangeAPI provides an API for querying the local HIBP data.
type RangeAPI struct {
	storage storage
}

// NewRangeAPI creates a new RangeAPI instance that can be used for querying k-proximity ranges.
// See the set of RangeAPIOption functions for customizing the behavior of the RangeAPI.
func NewRangeAPI(options ...RangeAPIOption) *RangeAPI {
	config := &queryConfig{
		commonConfig: commonConfig{
			dataDir: DefaultDataDir,
		},
	}

	for _, option := range options {
		option(config)
	}

	return &RangeAPI{
		storage: newFSStorage(config.dataDir, config.noCompression),
	}
}

// Query queries the local HIBP data for the given prefix.
// The function returns an io.ReadCloser that can be used to read the data, it should be closed as soon as possible
// to release the read lock on the file.
// It is the responsibility of the caller to close the returned io.ReadCloser.
func (q *RangeAPI) Query(prefix string) (io.ReadCloser, error) {
	reader, err := q.storage.LoadData(prefix)
	if err != nil {
		return nil, fmt.Errorf("loading data for prefix %q: %w", prefix, err)
	}

	return reader, nil
}

func readStateFile(stateFile io.ReadWriteSeeker) (int64, error) {
	state, err := io.ReadAll(stateFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}

		return 0, fmt.Errorf("reading state file: %w", err)
	}

	state = bytes.TrimSpace(state)

	if len(state) == 0 {
		return 0, nil
	}

	lastState, err := strconv.ParseInt(string(state), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing state file: %w", err)
	}

	if _, err := stateFile.Seek(0, io.SeekStart); err != nil {
		return 0, fmt.Errorf("seeking to beginning of state file: %w", err)
	}

	return lastState, nil
}

func wrapWithStateUpdate(startingState int64, stateFile io.ReadWriteSeeker, innerProgressFn ProgressFunc) ProgressFunc {
	return func(lowest, current, to, processed, remaining int64) error {
		err := func() error {
			if lowest < startingState+1000 && remaining > 0 {
				return nil
			}

			if _, err := stateFile.Seek(0, io.SeekStart); err != nil {
				return fmt.Errorf("seeking to beginning of state file: %w", err)
			}

			if _, err := stateFile.Write([]byte(fmt.Sprintf("%d", lowest))); err != nil {
				return fmt.Errorf("writing state file: %w", err)
			}

			startingState = lowest

			return nil
		}()

		if err != nil {
			fmt.Printf("updating state file: %v\n", err)
		}

		return innerProgressFn(lowest, current, to, processed, remaining)
	}
}
