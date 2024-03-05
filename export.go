package hibp

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

// The upstream Have-I-Been-Pwned API uses CRLF as line separator - so we are stuck with it,
// although it does not feel right.
var lineSeparator = []byte("\r\n")

func export(from, to int64, store storage, w io.Writer) error {
	for i := from; i < to; i++ {
		err := func() error {
			rangePrefix := toRangeString(i)

			dataReader, err := store.LoadData(rangePrefix)
			if err != nil {
				return fmt.Errorf("loading data for range %q: %w", rangePrefix, err)
			}
			defer dataReader.Close()

			lines, err := io.ReadAll(dataReader)
			if err != nil {
				return fmt.Errorf("reading data for range %q: %w", rangePrefix, err)
			}

			prefixedLines, err := prefixLines(lines, rangePrefix)
			if err != nil {
				return fmt.Errorf("prefixing lines for range %q: %w", rangePrefix, err)
			}

			if _, err := w.Write(prefixedLines); err != nil {
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
