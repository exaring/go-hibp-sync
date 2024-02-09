package hibpsync

import (
	"fmt"

	"github.com/alitto/pond"
	"github.com/hashicorp/go-retryablehttp"
)

const (
	defaultDataDir   = "./.hibp-data"
	defaultEndpoint  = "https://api.pwnedpasswords.com/range/"
	defaultCheckETag = true
	defaultWorkers   = 100
)

type syncConfig struct {
	dataDir   string
	endpoint  string
	checkETag bool
	worker    int
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

func WithWorkers(workers int) SyncOption {
	return func(c *syncConfig) {
		c.worker = workers
	}
}

func Sync(options ...SyncOption) error {
	config := &syncConfig{
		dataDir:   defaultDataDir,
		endpoint:  defaultEndpoint,
		checkETag: defaultCheckETag,
		worker:    defaultWorkers,
	}

	for _, option := range options {
		option(config)
	}

	rG, err := newRangeGenerator(0x00000, 0xFFFFF, "")
	if err != nil {
		return fmt.Errorf("creating range generator: %w", err)
	}

	retryClient := retryablehttp.NewClient() //TODO: add dnscache, timeout
	retryClient.RetryMax = 10
	retryClient.Logger = nil

	hc := hibpClient{
		endpoint:   config.endpoint,
		httpClient: retryClient.StandardClient(),
	}

	storage := fsStorage{
		dataDir: config.dataDir,
	}

	pool := pond.New(config.worker, 0, pond.MinWorkers(config.worker))
	defer pool.Stop()

	for {
		rangeIndex, ok, err := rG.Next()
		if err != nil {
			return fmt.Errorf("getting next range: %w", err)
		}

		if !ok {
			break
		}

		if rangeIndex%100 == 0 || rangeIndex < 10 {
			fmt.Printf("processing range %d\n", rangeIndex)
		}

		pool.Submit(func() {
			rangePrefix := toRangeString(rangeIndex)
			etag, err := storage.LoadETag(rangePrefix)
			if err != nil {
				fmt.Printf("error loading etag for range %q: %v\n", rangePrefix, err)
				return
			}

			resp, err := hc.RequestRange(rangePrefix, etag)
			if err != nil {
				fmt.Printf("error requesting range %q: %v\n", rangePrefix, err)
				return
			}

			if resp.NotModified {
				return
			}
			if err := storage.Save(rangePrefix, resp.ETag, resp.Data); err != nil {
				fmt.Printf("error saving range %q: %v\n", rangePrefix, err)
			}
		})
	}

	return nil
}
