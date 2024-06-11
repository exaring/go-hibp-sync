package hibp

import (
	"bytes"
	"go.uber.org/mock/gomock"
	"io"
	"math/rand"
	"testing"
)

func TestQuery(t *testing.T) {
	ctrl := gomock.NewController(t)
	storageMock := NewMockstorage(ctrl)

	storageMock.EXPECT().LoadData("00000").Return(io.NopCloser(bytes.NewReader([]byte("suffix:counter11\r\nsuffix:counter12"))), nil)

	i := HIBP{store: storageMock}

	reader, err := i.Query("00000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer reader.Close()

	lines, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// We expect the lines to not be prefixed with the range as this is what the response from the official
	// HIBP API looks like.
	if string(lines) != "suffix:counter11\r\nsuffix:counter12" {
		t.Fatalf("unexpected output: %q", string(lines))
	}
}

func BenchmarkQuery(b *testing.B) {
	const lastRange = 0x0000A

	dataDir := b.TempDir()

	h, err := New(WithDataDir(dataDir))
	if err != nil {
		b.Fatalf("initialising hibp sync: %v", err)
	}

	if err := h.Sync(SyncWithLastRange(lastRange)); err != nil {
		b.Fatalf("unexpected error: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		func() {
			rnd := rand.Intn(lastRange)

			reader, err := h.Query(toRangeString(int64(rnd)))
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
			defer func() {
				if reader.Close() != nil {
					b.Fatalf("unexpected error: %v", err)
				}
			}()

			data, err := io.ReadAll(reader)
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}

			b.SetBytes(int64(len(data)))
		}()
	}
}
