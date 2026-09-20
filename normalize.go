package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

// Normalizer reads CSV from a reader and writes a cleaned-up version to a
// writer. In strict mode it refuses to touch anything it isn't sure about
// and fails loudly instead. In lenient mode it repairs what it can.
type Normalizer struct {
	Delimiter rune
	Lenient   bool
}

func (n *Normalizer) Process(r io.Reader, w io.Writer) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}

	if n.Lenient {
		// A field opened with a quote but never closed makes Go's csv
		// reader consume the rest of the input looking for the closing
		// quote, then fail once it hits EOF. Closing it ourselves lets
		// the rest of a merely-truncated field/quote survive instead of
		// taking every row after it down with it.
		data = closeUnterminatedQuote(data, byte(n.Delimiter))
	}

	reader := csv.NewReader(bytes.NewReader(data))
	reader.Comma = n.Delimiter
	reader.LazyQuotes = n.Lenient
	if n.Lenient {
		// -1 disables Go's own field-count check so we can pad or
		// truncate ragged rows ourselves instead of bailing out.
		reader.FieldsPerRecord = -1
	}

	writer := csv.NewWriter(w)
	writer.Comma = n.Delimiter
	defer writer.Flush()

	var expected int
	line := 0

	for {
		record, err := reader.Read()
		line++
		if err == io.EOF {
			break
		}
		if err != nil {
			// Multi-line quoted fields make this line count approximate,
			// but it's close enough to point someone at the right row.
			if pe, ok := err.(*csv.ParseError); ok {
				return fmt.Errorf("line %d: %s", pe.StartLine, describeParseErr(pe))
			}
			return err
		}

		if n.Lenient {
			record = repairRecord(record)
			if isBlank(record) {
				continue
			}
			if expected == 0 {
				expected = len(record)
			}
			record = resize(record, expected)
		} else {
			if expected == 0 {
				expected = len(record)
			}
			for i, field := range record {
				if !utf8.ValidString(field) {
					return fmt.Errorf("line %d: field %d is not valid UTF-8 (use --lenient to repair it)", line, i+1)
				}
			}
		}

		if err := writer.Write(record); err != nil {
			return fmt.Errorf("line %d: writing output: %w", line, err)
		}
	}

	return writer.Error()
}

// closeUnterminatedQuote scans raw CSV bytes for a quoted field that is
// still open when the input ends and appends the missing closing quote.
// It only treats a quote as opening a field when it appears at the start
// of a field (right after the start of input, a delimiter, or a record
// break), mirroring how encoding/csv itself decides a field is quoted, so
// stray quotes elsewhere in a field are left alone.
func closeUnterminatedQuote(data []byte, delim byte) []byte {
	inQuotes := false
	atFieldStart := true

	for i := 0; i < len(data); i++ {
		c := data[i]
		if inQuotes {
			if c == '"' {
				if i+1 < len(data) && data[i+1] == '"' {
					i++ // escaped quote, still inside the field
					continue
				}
				inQuotes = false
				atFieldStart = false
			}
			continue
		}
		switch {
		case c == '"' && atFieldStart:
			inQuotes = true
			atFieldStart = false
		case c == delim || c == '\n':
			atFieldStart = true
		case c == '\r':
			// leave atFieldStart as-is; '\n' right after decides it
		default:
			atFieldStart = false
		}
	}

	if inQuotes {
		return append(data, '"')
	}
	return data
}

func repairRecord(record []string) []string {
	out := make([]string, len(record))
	for i, field := range record {
		field = strings.TrimSpace(field)
		if !utf8.ValidString(field) {
			field = strings.ToValidUTF8(field, "�")
		}
		out[i] = field
	}
	return out
}

// resize pads a short row with empty fields or truncates a long one so
// every row in the output has the same width as the first row we saw.
func resize(record []string, n int) []string {
	if len(record) == n {
		return record
	}
	if len(record) > n {
		return record[:n]
	}
	out := make([]string, n)
	copy(out, record)
	return out
}

func isBlank(record []string) bool {
	for _, field := range record {
		if field != "" {
			return false
		}
	}
	return true
}

func describeParseErr(pe *csv.ParseError) string {
	if pe.Err == csv.ErrFieldCount {
		return "wrong number of fields (use --lenient to pad or truncate rows)"
	}
	return pe.Err.Error()
}
