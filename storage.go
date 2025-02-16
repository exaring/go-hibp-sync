package hibp

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/klauspost/compress/zstd"
	"io"
	"os"
	"path"
	"strings"
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
	doNotUseCompression bool
	createDirsLock      syncPkg.Mutex
	lockMapLock         syncPkg.Mutex
	fileLocks           map[string]*syncPkg.RWMutex // prefix -> lock
}

var _ storage = (*fsStorage)(nil)

func newFSStorage(dataDir string, doNotUseCompression bool) *fsStorage {
	return &fsStorage{
		dataDir:             dataDir,
		doNotUseCompression: doNotUseCompression,
		fileLocks:           make(map[string]*syncPkg.RWMutex),
	}
}

type lockType int

const (
	read lockType = iota
	write
	tmpSuffix = ".tmp"
)

func (f *fsStorage) lockFile(key string, t lockType) func() {
	f.lockMapLock.Lock()
	fileLock, exists := f.fileLocks[key]
	if !exists {
		fileLock = &syncPkg.RWMutex{}

		// We cannot easily clean up the map of locks, because we would need to ensure nobody else is using
		// the lock at that moment.
		// This could be checked by trying to lock it, but would require us to lock fileLock while holding the
		// lockMapLock - effectively resulting in a global lock over all files.
		// Therefore, we consider it okay for this map to grow.
		// The upper limit is the number of files, which is 0xFFFFF (approx. 1 million files).
		// Additionally, this has the advantage of requiring fewer allocations and fewer objects to be gc'ed.
		f.fileLocks[key] = fileLock
	}
	f.lockMapLock.Unlock()

	if t == write {
		fileLock.Lock()
		return fileLock.Unlock
	}

	fileLock.RLock()
	return fileLock.RUnlock
}

func (f *fsStorage) Save(key, etag string, data []byte) error {
	key = strings.ToUpper(key)

	defer f.lockFile(key, write)()

	if err := f.createDirs(key); err != nil {
		return fmt.Errorf("creating data directory: %w", err)
	}

	filePath := f.filePath(key)
	filePathTmp := filePath + tmpSuffix

	// Creates the file if it doesn't exist, or truncates it if it does.
	// We use a temporary file to reduce the chance of corrupted files due to having to stop midair.
	// We do not have to check for remnants from a previous run because if there is a left-over temporary file, we will
	// overwrite it with the next run and rename it afterward.
	// Therefore, left-overs will be gone after the next run.
	file, err := os.Create(filePathTmp)
	if err != nil {
		return fmt.Errorf("creating file %q: %w", filePathTmp, err)
	}
	closeOnce := syncPkg.OnceValue(file.Close)
	defer closeOnce()

	var (
		w   io.Writer = file
		enc *zstd.Encoder
	)

	// We use the default compression level as non-scientific tests have shown that it's by far the best trade-off
	// between compression ratio and speed.
	if !f.doNotUseCompression {
		enc, err = zstd.NewWriter(file)
		if err != nil {
			return fmt.Errorf("creating zstd writer: %w", err)
		}
		defer enc.Close()

		w = enc
	}

	if _, err := w.Write([]byte(etag + "\n")); err != nil {
		return fmt.Errorf("writing etag to file %q: %w", filePathTmp, err)
	}

	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("writing data to file %q: %w", filePathTmp, err)
	}

	if err := file.Sync(); err != nil {
		return fmt.Errorf("syncing file %q to stable storage: %w", filePathTmp, err)
	}

	if enc != nil {
		if err := enc.Flush(); err != nil {
			return fmt.Errorf("flushing zstd writer: %w", err)
		}

		if err := enc.Close(); err != nil {
			return fmt.Errorf("closing zstd writer: %w", err)
		}
	}

	if err := closeOnce(); err != nil {
		return fmt.Errorf("closing file %q: %w", filePathTmp, err)
	}

	// Replaces an existing file; on unix-like systems that should be an atomic operation
	if err := os.Rename(filePathTmp, filePath); err != nil {
		return fmt.Errorf("renaming tmp file %q into actual file %q: %w", filePathTmp, filePath, err)
	}

	return nil
}

func (f *fsStorage) createDirs(key string) error {
	// We need to synchronize calls to Save because we don't want to create the same parent directory for several files
	// at the same time.
	// This could be made smarter to lock on a per-path basis, but that is most likely not worth the complexity.
	f.createDirsLock.Lock()
	defer f.createDirsLock.Unlock()

	return os.MkdirAll(f.subDir(key), dirMode)
}

func (f *fsStorage) LoadETag(key string) (string, error) {
	key = strings.ToUpper(key)

	defer f.lockFile(key, read)()

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
	callerWillCleanupResources := false

	key = strings.ToUpper(key)

	unlockFileFn := f.lockFile(key, read)
	defer func() {
		if !callerWillCleanupResources {
			unlockFileFn()
		}
	}()

	file, err := os.Open(f.filePath(key))
	if err != nil {
		return nil, fmt.Errorf("opening file %q: %w", f.filePath(key), err)
	}

	defer func() {
		if !callerWillCleanupResources {
			_ = file.Close()
		}
	}()

	var (
		r   io.Reader = file
		dec *zstd.Decoder
	)

	if !f.doNotUseCompression {
		dec, err = zstd.NewReader(file)
		if err != nil {
			return nil, fmt.Errorf("creating zstd reader: %w", err)
		}

		defer func() {
			if !callerWillCleanupResources {
				dec.Close()
			}
		}()

		r = dec
	}

	// Create a new buffered reader for efficient reading
	bufReader := bufio.NewReaderSize(r, 64*1024) // 64KB buffer - this should fit the whole file

	// Skip the first line containing the etag
	if _, _, err := bufReader.ReadLine(); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("skipping etag line in file %q: %w", f.filePath(key), err)
	}

	callerWillCleanupResources = true

	return &closableReader{
		Reader: bufReader,
		closeFn: func() error {
			defer unlockFileFn()
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
