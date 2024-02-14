package hibpsync

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	syncPkg "sync"
)

const (
	dirMode = 0o755 // TODO ???
)

type storage interface {
	Save(key, etag string, data []byte) error
	LoadETag(key string) (string, error)
	LoadData(key string) (io.ReadCloser, error)
}

type fsStorage struct {
	dataDir   string
	writeLock syncPkg.Mutex
}

var _ storage = (*fsStorage)(nil)

func (f *fsStorage) Save(key, etag string, data []byte) error {
	// We need to synchronize calls to Save because we don't want to create the same parent directory for several files
	// at the same time.
	f.writeLock.Lock()
	defer f.writeLock.Unlock()

	if err := os.MkdirAll(f.subDir(key), dirMode); err != nil {
		return fmt.Errorf("creating data directory: %w", err)
	}

	filePath := f.filePath(key)
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("creating file %q: %w", filePath, err)
	}
	defer file.Close()

	if _, err := file.WriteString(etag + "\n"); err != nil {
		return fmt.Errorf("writing etag to file %q: %w", filePath, err)
	}

	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("writing data to file %q: %w", filePath, err)
	}

	return nil
}

func (f *fsStorage) LoadETag(key string) (string, error) {
	file, err := os.Open(f.filePath(key))
	if err != nil {
		return "", fmt.Errorf("opening file %q: %w", f.filePath(key), err)
	}
	defer file.Close()

	etag, err := bufio.NewReader(file).ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("reading etag from file %q: %w", f.filePath(key), err)
	}

	// Remove the newline character from the etag
	return etag[:len(etag)-1], nil
}

func (f *fsStorage) LoadData(key string) (io.ReadCloser, error) {
	file, err := os.Open(f.filePath(key))
	if err != nil {
		return nil, fmt.Errorf("opening file %q: %w", f.filePath(key), err)
	}

	// Create a new buffered reader for efficient reading
	bufReader := bufio.NewReaderSize(file, 64*1024) // 64KB buffer - this should fit the whole file

	// Skip the first line containing the etag
	if _, _, err := bufReader.ReadLine(); err != nil && !errors.Is(err, io.EOF) {
		defer file.Close()

		return nil, fmt.Errorf("skipping etag line in file %q: %w", f.filePath(key), err)
	}

	return &closableReader{
		Reader: bufReader,
		Closer: file,
	}, nil
}

func (f *fsStorage) subDir(key string) string {
	subDir := key[:2]
	return path.Join(f.dataDir, subDir)
}

func (f *fsStorage) filePath(key string) string {
	return path.Join(f.subDir(key), key[2:])
}

type closableReader struct {
	io.Reader
	io.Closer
}
