package hibpsync

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/alitto/pond"
	"github.com/hashicorp/go-retryablehttp"
	"io"
	"os"
	"strconv"
)

const (
	defaultDataDir   = "./.hibp-data"
	defaultEndpoint  = "https://api.pwnedpasswords.com/range/"
	defaultWorkers   = 50
	DefaultStateFile = "./.hibp-data/state"
	defaultLastRange = 0xFFFFF
)

type ProgressFunc func(lowest, current, to, processed, remaining int64) error

func Sync(options ...SyncOption) error {
	config := &syncConfig{
		commonConfig: commonConfig{
			dataDir: defaultDataDir,
		},
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

	storage := &fsStorage{
		dataDir:             config.dataDir,
		doNotUseCompression: config.noCompression,
	}

	pool := pond.New(config.minWorkers, 0, pond.MinWorkers(config.minWorkers))

	return sync(from, config.lastRange+1, client, storage, pool, config.progressFn)
}

func Export(w io.Writer, options ...ExportOption) error {
	config := &exportConfig{
		commonConfig: commonConfig{
			dataDir: defaultDataDir,
		},
	}

	for _, option := range options {
		option(config)
	}

	storage := &fsStorage{
		dataDir:             config.dataDir,
		doNotUseCompression: config.noCompression,
	}

	return export(0, defaultLastRange+1, storage, w)
}

type RangeAPI struct {
	storage storage
}

func NewRangeAPI(options ...QueryOption) *RangeAPI {
	config := &queryConfig{
		commonConfig: commonConfig{
			dataDir: defaultDataDir,
		},
	}

	for _, option := range options {
		option(config)
	}

	return &RangeAPI{
		storage: &fsStorage{
			dataDir:             config.dataDir,
			doNotUseCompression: config.noCompression,
		},
	}
}

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
