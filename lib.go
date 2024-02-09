package hibpsync

const (
	defaultDataDir   = "./.hibp-data"
	defaultEndpoint  = "https://api.pwnedpasswords.com/range/"
	defaultCheckETag = true
)

type syncConfig struct {
	dataDir   string
	endpoint  string
	checkETag bool
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

func Sync(options ...SyncOption) {
	config := &syncConfig{
		dataDir:   defaultDataDir,
		endpoint:  defaultEndpoint,
		checkETag: defaultCheckETag,
	}

	for _, option := range options {
		option(config)
	}

	// TODO: Implement sync
	// We want to use a pool of workers that draw their range from
}
