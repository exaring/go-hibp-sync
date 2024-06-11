package hibp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/alitto/pond"
	"github.com/hashicorp/go-retryablehttp"
	"io"
	"os"
	"path"
	"strconv"
	"sync/atomic"
	"time"
)

const (
	DefaultDataDir                   = "./.hibp-data"
	DefaultStateFileName             = "state"
	defaultEndpoint                  = "https://api.pwnedpasswords.com/range/"
	defaultWorkers                   = 50
	defaultLastRange                 = 0xFFFFF
	hibpMostRecentSuccessfulSyncPath = ".most_recent_successful_sync"
)

// HIBP bundles the functionality of the HIBP package.
// In order to allow concurrent operations on the local, file-based dataset efficiently and safely, a shared set of
// locks is required - this gets managed by the HIBP type.
type HIBP struct {
	store                    storage
	dataDir                  string
	mostRecentSuccessfulSync atomic.Pointer[time.Time]
}

func New(options ...CommonOption) (*HIBP, error) {
	config := commonConfig{
		dataDir:       DefaultDataDir,
		noCompression: false,
	}

	for _, option := range options {
		option(&config)
	}

	storage := newFSStorage(config.dataDir, config.noCompression)

	var mostRecentSuccessfulSync time.Time

	mostRecentSuccessfulSyncPath := path.Join(config.dataDir, hibpMostRecentSuccessfulSyncPath)
	mostRecentSuccessfullSyncBytes, err := os.ReadFile(mostRecentSuccessfulSyncPath)
	if err != nil {
		// It is ok if the file does not exist
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("reading timestamp of most recent successful sync at %q: %w", mostRecentSuccessfulSyncPath, err)
		}
	} else {
		seconds, err := strconv.ParseInt(string(mostRecentSuccessfullSyncBytes), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing timestamp %q of most recent successful sync from %q: %w", mostRecentSuccessfullSyncBytes, mostRecentSuccessfulSyncPath, err)
		}

		mostRecentSuccessfulSync = time.Unix(seconds, 0)
	}

	h := &HIBP{
		store:   storage,
		dataDir: config.dataDir,
	}

	h.mostRecentSuccessfulSync.Store(&mostRecentSuccessfulSync)

	return h, nil
}

// Sync copies the ranges, i.e., the HIBP data, from the upstream API to the local storage.
// The function will start from the lowest prefix and continue until the highest prefix.
// See the set of SyncOption functions for customizing the behavior of the sync operation.
func (h *HIBP) Sync(options ...SyncOption) error {
	config := &syncConfig{
		ctx:                                 context.Background(),
		endpoint:                            defaultEndpoint,
		minWorkers:                          defaultWorkers,
		progressFn:                          func(_, _, _, _, _ int64) error { return nil },
		lastRange:                           defaultLastRange,
		trackMostRecentSuccessfulSyncInFile: true,
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

	// It is important to create a non-buffering/blocking pool because we don't want to schedule all jobs upfront.
	// This would cause problems, especially when cancelling the context.
	pool := pond.New(config.minWorkers, 0, pond.MinWorkers(config.minWorkers))

	if err := sync(config.ctx, from, config.lastRange+1, client, h.store, pool, config.progressFn); err != nil {
		return err
	}

	now := time.Now()
	h.mostRecentSuccessfulSync.Store(&now)

	if config.trackMostRecentSuccessfulSyncInFile {
		mostRecentSuccessfulSyncPath := path.Join(h.dataDir, hibpMostRecentSuccessfulSyncPath)

		if err := os.WriteFile(mostRecentSuccessfulSyncPath, []byte(strconv.FormatInt(now.Unix(), 10)), 0o644); err != nil {
			return fmt.Errorf("writing timestamp of most recent successful sync to %q: %w", mostRecentSuccessfulSyncPath, err)
		}
	}

	return nil
}

// Export writes the dataset to the given writer.
// The data is written as a continuous stream with no indication of the "prefix boundaries",
// the format therefore differs from the official Have-I-Been-Pwned API and from `Query`, which is mimicking the API.
// Lines have the schema "<prefix><suffix>:<count>".
func (h *HIBP) Export(w io.Writer) error {
	return export(0, defaultLastRange+1, h.store, w)
}

// Query queries the local dataset for the given prefix.
// The function returns an io.ReadCloser that can be used to read the data, it should be closed as soon as possible
// to release the read lock on the file.
// It is the responsibility of the caller to close the returned io.ReadCloser.
// The resulting lines do NOT start with the prefix, they are following the schema "<suffix>:<count>".
// This is equivalent to the response of the official Have-I-Been-Pwned API.
func (h *HIBP) Query(prefix string) (io.ReadCloser, error) {
	reader, err := h.store.LoadData(prefix)
	if err != nil {
		return nil, fmt.Errorf("loading data for prefix %q: %w", prefix, err)
	}

	return reader, nil
}

// MostRecentSuccessfulSync returns the point in the most recent successful sync finished.
func (h *HIBP) MostRecentSuccessfulSync() time.Time {
	return *h.mostRecentSuccessfulSync.Load()
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
