package hibpsync

import (
	"fmt"
	"io"
)

var lineSeparator = []byte("\n")

func export(from, to int64, store storage, w io.Writer) error {
	for i := from; i < to; i++ {
		err := func() error {
			rangePrefix := toRangeString(i)

			dataReader, err := store.LoadData(rangePrefix)
			if err != nil {
				return fmt.Errorf("loading data for range %q: %w", rangePrefix, err)
			}
			defer dataReader.Close()

			if _, err := io.Copy(w, dataReader); err != nil {
				return fmt.Errorf("writing data for range %q: %w", rangePrefix, err)
			}

			if i+1 < to {
				if _, err := w.Write(lineSeparator); err != nil {
					return fmt.Errorf("writing line separator to export writer: %w", err)
				}
			}

			return nil
		}()

		if err != nil {
			return err
		}
	}

	if closer, ok := w.(io.Closer); ok {
		return closer.Close()
	}

	return nil
}
