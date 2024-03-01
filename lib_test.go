package hibp

import (
	"io"
	"math/rand"
	"testing"
)

func BenchmarkQuery(b *testing.B) {
	const lastRange = 0x0000A

	dataDir := b.TempDir()

	h := New(WithDataDir(dataDir))

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
