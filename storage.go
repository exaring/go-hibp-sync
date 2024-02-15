package hibpsync

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/klauspost/compress/zstd"
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
	dataDir             string
	writeLock           syncPkg.Mutex
	doNotUseCompression bool
}

var _ storage = (*fsStorage)(nil)

func (f *fsStorage) Save(key, etag string, data []byte) error {
	if err := f.createDirs(key); err != nil {
		return fmt.Errorf("creating data directory: %w", err)
	}

	filePath := f.filePath(key)

	// Creates the file if it doesn't exist, or truncates it if it does.
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("creating file %q: %w", filePath, err)
	}
	defer file.Close()

	var w io.Writer = file

	// We use the default compression level as non-scientific tests have shown that it's by far the best trade-off
	// between compression ratio and speed.
	if !f.doNotUseCompression {
		enc, err := zstd.NewWriter(file)
		if err != nil {
			return fmt.Errorf("creating zstd writer: %w", err)
		}
		defer enc.Close()

		w = enc
	}

	if _, err := w.Write([]byte(etag + "\n")); err != nil {
		return fmt.Errorf("writing etag to file %q: %w", filePath, err)
	}

	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("writing data to file %q: %w", filePath, err)
	}

	return nil
}

func (f *fsStorage) createDirs(key string) error {
	// We need to synchronize calls to Save because we don't want to create the same parent directory for several files
	// at the same time.
	f.writeLock.Lock()
	defer f.writeLock.Unlock()

	return os.MkdirAll(f.subDir(key), dirMode)
}

func (f *fsStorage) LoadETag(key string) (string, error) {
	file, err := os.Open(f.filePath(key))
	if err != nil {
		return "", fmt.Errorf("opening file %q: %w", f.filePath(key), err)
	}
	defer file.Close()

	var r io.Reader = file

	if !f.doNotUseCompression {
		dec, err := zstd.NewReader(file)
		if err != nil {
			return "", fmt.Errorf("creating zstd reader: %w", err)
		}
		defer dec.Close()

		r = dec
	}

	etag, err := bufio.NewReader(r).ReadString('\n')
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

	var (
		r   io.Reader = file
		dec *zstd.Decoder
	)

	if !f.doNotUseCompression {
		dec, err = zstd.NewReader(file)
		if err != nil {
			return nil, fmt.Errorf("creating zstd reader: %w", err)
		}

		r = dec
	}

	// Create a new buffered reader for efficient reading
	bufReader := bufio.NewReaderSize(r, 64*1024) // 64KB buffer - this should fit the whole file

	// Skip the first line containing the etag
	if _, _, err := bufReader.ReadLine(); err != nil && !errors.Is(err, io.EOF) {
		defer file.Close()
		if dec != nil {
			defer dec.Close()
		}

		return nil, fmt.Errorf("skipping etag line in file %q: %w", f.filePath(key), err)
	}

	return &closableReader{
		Reader: bufReader,
		closeFn: func() error {
			defer file.Close()
			if dec != nil {
				defer dec.Close()
			}

			return nil
		},
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
	closeFn func() error
}

func (c *closableReader) Close() error {
	return c.closeFn()
}
