package hibpsync

import (
	"io"
	"math/rand"
	"testing"
)

func BenchmarkQuery(b *testing.B) {
	const lastRange = 0x0000A

	dataDir := b.TempDir()

	if err := Sync(WithDataDir(dataDir), WithLastRange(lastRange)); err != nil {
		b.Fatalf("unexpected error: %v", err)
	}

	querier := NewRangeAPI(dataDir, true)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		func() {
			rnd := rand.Intn(lastRange)

			reader, err := querier.Query(toRangeString(int64(rnd)))
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
			defer reader.Close()

			data, err := io.ReadAll(reader)
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}

			b.SetBytes(int64(len(data)))
		}()
	}
}
