package hibpsync

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sync"
)

const (
	fileMode = 0666 // TODO ???
	dirMode  = 0744 // TODO ???
)

type fsStorage struct {
	dataDir   string
	writeLock sync.Mutex
}

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
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
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
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("opening file %q: %w", f.filePath(key), err)
	}

	if err := skipLine(file); err != nil {
		file.Close()
		return nil, fmt.Errorf("skipping etag line in file %q: %w", f.filePath(key), err)
	}

	return file, nil
}

func skipLine(reader io.ReadSeeker) error {
	// Create a new buffered reader for efficient reading
	br := bufio.NewReader(reader)

	// Read until the first newline character
	_, err := br.ReadString('\n')
	if err != nil && err != io.EOF {
		return err
	}

	// Get the current offset
	offset, err := reader.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}

	// Seek back to the beginning of the file
	_, err = reader.Seek(offset, io.SeekStart)
	if err != nil {
		return err
	}

	return nil
}

func (f *fsStorage) subDir(key string) string {
	subDir := key[:2]
	return path.Join(f.dataDir, subDir)
}

func (f *fsStorage) filePath(key string) string {
	return path.Join(f.subDir(key), key[2:])
}
