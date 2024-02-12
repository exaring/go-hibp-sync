package hibpsync

import (
	"fmt"
	mapset "github.com/deckarep/golang-set/v2"
	"math"
	"sync"
	"sync/atomic"
)

type ProgressFunc func(lowest, current, to int64) error

type rangeGenerator struct {
	from, to       int64
	idx            *atomic.Int64
	inFlightSet    mapset.Set[int64]
	onProgress     ProgressFunc
	onProgressLock sync.Mutex
}

func newRangeGenerator(from, to int64, onProgress ProgressFunc) *rangeGenerator {
	idx := &atomic.Int64{}
	idx.Store(from)

	return &rangeGenerator{
		from:        from,
		to:          to,
		idx:         idx,
		inFlightSet: mapset.NewSet[int64](),
		onProgress:  onProgress,
	}
}

func (r *rangeGenerator) Next(fn func(r int64) error) (bool, error) {
	current := r.idx.Add(1) - 1

	if current >= r.to {
		return false, nil
	}

	r.inFlightSet.Add(current)

	if err := fn(current); err != nil {
		return false, err
	}

	r.inFlightSet.Remove(current)

	if current%10 == 0 || current == r.to-1 {
		r.onProgressLock.Lock()
		defer r.onProgressLock.Unlock()

		// TODO: Compute remaining and provide to progress function

		if err := r.onProgress(r.lowestInFlight(), current, r.to); err != nil {
			return false, err
		}
	}

	return true, nil
}

func (r *rangeGenerator) lowestInFlight() int64 {
	lowest := int64(math.MaxInt64)

	for _, a := range r.inFlightSet.ToSlice() {
		lowest = min(lowest, a)
	}

	return lowest
}

func toRangeString(i int64) string {
	return fmt.Sprintf("%05X", i)
}
