package hibpsync

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/alitto/pond"
	mapset "github.com/deckarep/golang-set/v2"
	"math"
	syncPkg "sync"
	"sync/atomic"
)

func sync(ctx context.Context, from, to int64, client *hibpClient, store storage, pool *pond.WorkerPool, onProgress ProgressFunc) error {
	var (
		mErr           error
		errLock        syncPkg.Mutex
		processed      atomic.Int64
		inFlightSet    = mapset.NewSet[int64]()
		onProgressLock syncPkg.Mutex
	)

	processed.Store(from)

	for i := from; i < to; i++ {
		current := i

		// Pool is configured to be non-buffering, i.e., when the context gets canceled, we will finish the jobs
		// that are currently being processed, but we will not start new ones.
		if err := ctx.Err(); err != nil {
			return err
		}

		pool.Submit(func() {
			rangePrefix := toRangeString(current)

			err := func() error {
				inFlightSet.Add(current)

				// We basically ignore any error here because we can still process the range even if we can't load the etag
				etag, err := store.LoadETag(rangePrefix)
				if err != nil {
					etag = ""
				}

				resp, err := client.RequestRange(rangePrefix, etag)
				if err != nil {
					return err
				}

				if !resp.NotModified {
					prefixedLines, err := prefixLines(resp.Data, rangePrefix)
					if err != nil {
						return fmt.Errorf("prefixing lines: %w", err)
					}

					if err := store.Save(rangePrefix, resp.ETag, prefixedLines); err != nil {
						return fmt.Errorf("saving range: %w", err)
					}
				}

				p := processed.Add(1)

				inFlightSet.Remove(current)

				lowest := lowestInFlight(inFlightSet, to)
				remaining := to - p

				if p%10 == 0 || remaining == 0 {
					onProgressLock.Lock()
					defer onProgressLock.Unlock()

					if err := onProgress(lowest, current, to, p, remaining); err != nil {
						return fmt.Errorf("reporting progress: %w", err)
					}
				}

				return nil
			}()

			if err != nil {
				errLock.Lock()
				defer errLock.Unlock()

				mErr = errors.Join(mErr, fmt.Errorf("processing range %q: %w", rangePrefix, err))
			}
		})
	}

	pool.StopAndWait()

	return mErr
}

func toRangeString(i int64) string {
	return fmt.Sprintf("%05X", i)
}

func prefixLines(in []byte, prefix string) ([]byte, error) {
	firstLine := true

	// Actually, we know that the size will be: len(in) + rows * len(prefix)
	// But we do not know the number of rows - so starting from len(in) seems to be a good choice.
	out := bytes.NewBuffer(make([]byte, 0, len(in)))

	scanner := bufio.NewScanner(bytes.NewReader(in))
	for scanner.Scan() {
		if !firstLine {
			if _, err := out.Write(lineSeparator); err != nil {
				return nil, fmt.Errorf("adding line separator: %w", err)
			}
		}

		firstLine = false

		if _, err := out.Write([]byte(prefix)); err != nil {
			return nil, fmt.Errorf("adding prefix: %w", err)
		}

		if _, err := out.Write(scanner.Bytes()); err != nil {
			return nil, fmt.Errorf("adding suffix and counter: %w", err)
		}
	}

	return out.Bytes(), nil
}

func lowestInFlight(inFlight mapset.Set[int64], to int64) int64 {
	lowest := int64(math.MaxInt64)

	for _, a := range inFlight.ToSlice() {
		lowest = min(lowest, a)
	}

	if lowest == math.MaxInt64 {
		return to - 1
	}

	return lowest
}
