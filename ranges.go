package hibpsync

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"sync"
)

const writeStateEveryN = 10

type rangeGenerator struct {
	idx, to       int
	lock          sync.Mutex
	stateFilePath string
}

func newRangeGenerator(from, to int, stateFilePath string) (*rangeGenerator, error) {
	// Check if the state file exists and read the last state from it.
	// This is useful to resume the sync process after a crash.
	if stateFilePath != "" {
		bytez, err := os.ReadFile(stateFilePath)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("reading state file: %w", err)
		}

		from, err = strconv.Atoi(string(bytez))
		if err != nil {
			return nil, fmt.Errorf("parsing state file: %w", err)
		}
	}

	return &rangeGenerator{
		idx:           from,
		to:            to,
		stateFilePath: stateFilePath,
	}, nil
}

func (r *rangeGenerator) Next() (int, bool, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.idx > r.to {
		return 0, false, nil
	}

	current := r.idx
	r.idx++

	if r.stateFilePath != "" && (current%writeStateEveryN == 0 || current == r.to) {
		if err := os.WriteFile(r.stateFilePath, []byte(fmt.Sprintf("%d", current)), 0644); err != nil {
			return 0, false, fmt.Errorf("writing state file: %w", err)
		}
	}

	return current, true, nil
}

func toRangeString(i int) string {
	return fmt.Sprintf("%05X", i)
}
