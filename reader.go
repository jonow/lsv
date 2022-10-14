package lsv

import (
	"bufio"
	"errors"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Error messages.
var (
	ErrNoClosingRaw = errors.New("raw literal not closed")
	errInvalidDelim = errors.New("invalid comment, raw, or escape delimiter")
)

// validDelim determines if the rune cast be used by the reader.
func validDelim(r rune) bool {
	return r != 0 && !unicode.IsSpace(r) && utf8.ValidRune(r) && r != utf8.RuneError
}

// Reader reads values from a LSV-encoded file.
type Reader struct {
	p Parameters

	r *bufio.Reader
}

// NewReader returns a new Reader that reads from r.
func NewReader(r io.Reader) *Reader {
	return &Reader{
		p: DefaultParameters(),
		r: bufio.NewReader(r),
	}
}

// NewCustomReader returns a new Reader that reads from r with custom LSV
// parameters.
func NewCustomReader(r io.Reader, p Parameters) *Reader {
	return &Reader{
		p: p,
		r: bufio.NewReader(r),
	}
}

// ReadAll reads all the remaining values from r. A successful call returns
// err == nil, not err == io.EOF. Because ReadAll is defined to read until EOF,
// it does not treat end of file as an error to be reported.
func (r *Reader) ReadAll() ([]string, error) {
	var values []string

	for {
		value, err := r.readValue()
		if err == io.EOF {
			return values, nil
		}
		if err != nil {
			return nil, err
		}

		values = append(values, value)
	}
}

// Read reads one value from r. If a raw string literal is started but not
// closed, Read returns ErrNoClosingRaw. If there is no data left to be read,
// Read returns io.EOF.
func (r *Reader) Read() (string, error) {
	return r.readValue()
}

// readLine reads the next line (with the trailing end-line). If some bytes were
// read, then the error is never io.EOF. The result is only valid until the next
// call to readLine.
func (r *Reader) readLine() (string, error) {
	line, err := r.r.ReadString('\n')

	// If bytes are read, do not return EOF
	if len(line) > 0 && err == io.EOF {
		err = nil
	}

	return line, err
}

// readValue is the internal helper function for Read.
func (r *Reader) readValue() (string, error) {
	if r.p.Comment == r.p.Raw || r.p.Comment == r.p.Escape ||
		r.p.Raw == r.p.Escape || !validDelim(r.p.Comment) ||
		!validDelim(r.p.Raw) || !validDelim(r.p.Escape) {
		return "", errInvalidDelim
	}

	var inRaw bool
	var line string
	var rawString strings.Builder
	var err error

	for {
		line, err = r.readLine()
		if err != nil {
			break
		}

		if !inRaw {
			// Trim leading whitespace if not in raw string literal
			if r.p.TrimLeadingSpace {
				line = strings.TrimLeftFunc(line, unicode.IsSpace)
			}

			// Skip empty lines or lines with only whitespace
			if line == "" {
				continue
			}

			// Check if the value is a raw string literal
			if c, size := utf8.DecodeRuneInString(line); c == r.p.Raw {
				inRaw = true
				line = line[size:]
			}
		}

		// Trim any comment not in raw string
		line = r.p.trimComment(line, inRaw)
		if line != "" {
			if inRaw {
				// If in raw string literal, add to rawString instead of
				// returning the value so the rest of the value can be read

				var last, prev1, prev2 rune
				var j, k int
				for i := len(line); i > 0; {
					char, size := utf8.DecodeLastRuneInString(line[0:i])
					i -= size
					if !unicode.IsSpace(char) && last == 0 {
						last = char
						j = i
					} else if last != 0 && prev1 == 0 {
						prev1 = char
						k = i
					} else if last != 0 && prev1 != 0 && prev2 == 0 {
						prev2 = char
						break
					}
				}

				if r.p.isRaw(last, prev1, prev2) {
					rawString.WriteString(line[:j])
					line = rawString.String()
					rawString.Reset()
					inRaw = false
					break
				} else if last == r.p.Raw && prev1 == r.p.Escape {
					// Trim escape character
					line = line[:k] + line[j:]
				}
				rawString.WriteString(line)
			} else {
				// Trim trailing whitespace
				line = strings.TrimRightFunc(line, unicode.IsSpace)

				// Replace escaped comments with comment character
				line = strings.ReplaceAll(line,
					string(r.p.Escape)+string(r.p.Comment), string(r.p.Comment))

				if len(line) > 0 {
					break
				}
			}
		}
	}

	if inRaw {
		return "", ErrNoClosingRaw
	} else if err != nil {
		return "", err
	}

	return line, nil
}
