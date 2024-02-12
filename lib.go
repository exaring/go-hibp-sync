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
	"sync"
)

const (
	defaultDataDir   = "./.hibp-data"
	defaultEndpoint  = "https://api.pwnedpasswords.com/range/"
	defaultCheckETag = true
	defaultWorkers   = 50
	DefaultStateFile = "./.hibp-data/state"
)

type syncConfig struct {
	dataDir    string
	endpoint   string
	checkETag  bool
	minWorkers int
	progressFn ProgressFunc
	stateFile  io.ReadWriteSeeker
}

type SyncOption func(*syncConfig)

func WithDataDir(dataDir string) SyncOption {
	return func(c *syncConfig) {
		c.dataDir = dataDir
	}
}

func WithEndpoint(endpoint string) SyncOption {
	return func(c *syncConfig) {
		c.endpoint = endpoint
	}
}

func WithCheckETag(checkETag bool) SyncOption {
	return func(c *syncConfig) {
		c.checkETag = checkETag
	}
}

func WithMinWorkers(workers int) SyncOption {
	return func(c *syncConfig) {
		c.minWorkers = workers
	}
}

func WithStateFile(stateFile io.ReadWriteSeeker) SyncOption {
	return func(c *syncConfig) {
		c.stateFile = stateFile
	}
}

func WithProgressFn(progressFn ProgressFunc) SyncOption {
	return func(c *syncConfig) {
		c.progressFn = progressFn
	}
}

func Sync(options ...SyncOption) error {
	config := &syncConfig{
		dataDir:    defaultDataDir,
		endpoint:   defaultEndpoint,
		checkETag:  defaultCheckETag,
		minWorkers: defaultWorkers,
		progressFn: func(_, _, _ int64) error { return nil },
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
		innerProgressFn := config.progressFn

		config.progressFn = func(lowest, current, to int64) error {
			err := func() error {
				if lowest < lastState+1000 {
					return nil
				}

				if _, err := config.stateFile.Seek(0, io.SeekStart); err != nil {
					return fmt.Errorf("seeking to beginning of state file: %w", err)
				}

				if _, err := config.stateFile.Write([]byte(fmt.Sprintf("%d", lowest))); err != nil {
					return fmt.Errorf("writing state file: %w", err)
				}

				lastState = lowest

				return nil
			}()

			if err != nil {
				fmt.Printf("updating state file: %v\n", err)
			}

			return innerProgressFn(lowest, current, to)
		}
	}

	rG := newRangeGenerator(from, 0xFFFFF+1, config.progressFn)

	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 10
	retryClient.Logger = nil

	hc := hibpClient{
		endpoint:   config.endpoint,
		httpClient: retryClient.StandardClient(),
		maxRetries: 2,
	}

	storage := fsStorage{
		dataDir: config.dataDir,
	}

	pool := pond.New(config.minWorkers, 0, pond.MinWorkers(config.minWorkers))
	defer pool.Stop()

	var (
		outerErr error
		errLock  sync.Mutex
	)

	for !pool.Stopped() {
		pool.Submit(func() {
			keepGoing, err := rG.Next(func(r int64) error {
				rangePrefix := toRangeString(r)

				etag, _ := storage.LoadETag(rangePrefix)
				// TODO: Log error with debug level

				resp, err := hc.RequestRange(rangePrefix, etag)
				if err != nil {
					return fmt.Errorf("error requesting range %q: %w", rangePrefix, err)
				}

				if resp.NotModified {
					return nil
				}

				if err := storage.Save(rangePrefix, resp.ETag, resp.Data); err != nil {
					return fmt.Errorf("error saving range %q: %w", rangePrefix, err)
				}

				return nil
			})
			if err != nil {
				errLock.Lock()
				defer errLock.Unlock()

				outerErr = errors.Join(fmt.Errorf("processing range: %w", err))
			}

			if !keepGoing {
				pool.Stop()
			}
		})
	}

	return outerErr
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
