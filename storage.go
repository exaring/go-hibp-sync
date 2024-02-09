package hibpsync

import "sync"

type fsStorage struct {
	dataDir   string
	writeLock sync.Mutex
}

func (f *fsStorage) Save(key, etag string, data []byte) error {
	// We need to synchronize calls to Save because we don't want to create the same parent directory for several files
	// at the same time.
	f.writeLock.Lock()
	defer f.writeLock.Unlock()

	// TODO: Implement Save

	return nil
}

func (f *fsStorage) LoadETag(key string) (string, error) {
	// TODO: Implement LoadETag

	return "", nil
}

func (f *fsStorage) LoadData(key string) ([]byte, error) {
	// TODO: Implement LoadData

	return nil, nil
}
