package hibp

import (
	"bytes"
	"github.com/klauspost/compress/zstd"
	"io"
	"os"
	"strings"
	"testing"
)

func TestFSStorage(t *testing.T) {
	testWriteRead := func(t *testing.T, useCompression bool) {
		key := "key"

		tmpDir := t.TempDir()

		storage := newFSStorage(tmpDir, !useCompression)

		err := storage.Save(key, "etag", []byte("data"))
		if err != nil {
			t.Fatalf("could not write: %v", err)
		}

		// First, let's check the raw file
		var reader io.Reader

		// We have to explicitly refer to the upper-cased key as Save/Load perform this operation internally.
		// Depending on the file system that does make a difference.
		file, err := os.Open(storage.filePath(strings.ToUpper(key)))
		if err != nil {
			t.Fatalf("could not open file: %v", err)
		}
		defer file.Close()

		reader = file

		if useCompression {
			dec, err := zstd.NewReader(file)
			if err != nil {
				t.Fatalf("could not create decompressor: %v", err)
			}

			defer dec.Close()

			reader = dec
		}

		raw, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("could not read file: %v", err)
		}

		if !bytes.Equal([]byte("etag\ndata"), raw) {
			t.Fatalf("unexpected data: %q", raw)
		}

		// Then, let's check the API
		etag, err := storage.LoadETag(key)
		if err != nil {
			t.Fatalf("could not read etag: %v", err)
		}

		if etag != "etag" {
			t.Fatalf("unexpected etag: %q", etag)
		}

		readCloser, err := storage.LoadData(key)
		if err != nil {
			t.Fatalf("could not open reader: %v", err)
		}
		defer readCloser.Close()

		all, err := io.ReadAll(readCloser)
		if err != nil {
			t.Fatalf("could not read file: %v", err)
		}

		if !bytes.Equal([]byte("data"), all) {
			t.Fatalf("unexpected data: %q", all)
		}

	}

	t.Run("write and read data without compression", func(t *testing.T) {
		testWriteRead(t, false)
	})

	t.Run("write and read data with compression", func(t *testing.T) {
		testWriteRead(t, true)
	})
}
