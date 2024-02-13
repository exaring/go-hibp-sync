package hibpsync

import (
	"bufio"
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

			if err := copyAndPrefixLines(dataReader, w, rangePrefix); err != nil {
				return fmt.Errorf("copying data for range %q: %w", rangePrefix, err)
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

func copyAndPrefixLines(in io.Reader, out io.Writer, prefix string) error {
	firstLine := true

	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		if !firstLine {
			if _, err := out.Write(lineSeparator); err != nil {
				return fmt.Errorf("writing line separator to export writer: %w", err)
			}
		}

		firstLine = false

		if _, err := out.Write([]byte(prefix)); err != nil {
			return fmt.Errorf("writing prefix to export writer: %w", err)
		}

		if _, err := out.Write(scanner.Bytes()); err != nil {
			return fmt.Errorf("writing suffix and count to export writer: %w", err)
		}
	}

	return nil
}
