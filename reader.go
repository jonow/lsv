package lsv

import (
	"bufio"
	"errors"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

// TODO: Add support for empty entries

const (
	defaultComment = '#'
	defaultRaw     = '"'
	defaultEscape  = '\\'
)

var ErrNoClosingRaw = errors.New("raw literal not closed")

var errInvalidDelim = errors.New("invalid comment, raw, or escape delimiter")

// validDelim determines if the rune cast be used by the reader.
func validDelim(r rune) bool {
	return r != 0 && !unicode.IsSpace(r) && utf8.ValidRune(r) && r != utf8.RuneError
}

// Reader reads values from a LSV-encoded file.
type Reader struct {
	// Comment is the comment character. Any characters following the comment
	// until the next newline, including leading and trailing whitespace, are
	// ignored. It must be a valid rune that is not whitespace. Comment
	// characters preceded by the Escape rune are unescaped and treated as
	// values.
	Comment rune

	// Raw is the character that indicates the start and end of a raw literal
	// and can only appear as the first or last non-whitespace character on a
	// line. Any text contained between two Raw characters is considered a
	// value. That is, all Comment, Raw, and Escape characters are part of the
	// value except if the closing Raw character is escaped.
	Raw rune

	// Escape is the character used to indicate that a Comment or Raw character
	// is part of the value. If either of these characters are escaped, the
	// escape character is stripped and the original character is maintained.
	// An Escape character can be escaped itself.
	Escape rune

	// If TrimLeadingSpace is true, leading white space in a field is ignored.
	// This is true by default.
	TrimLeadingSpace bool

	r *bufio.Reader
}

// NewReader returns a new Reader that reads from r.
func NewReader(r io.Reader) *Reader {
	return &Reader{
		Comment:          defaultComment,
		Raw:              defaultRaw,
		Escape:           defaultEscape,
		TrimLeadingSpace: true,
		r:                bufio.NewReader(r),
	}
}

// Read reads one value from r. If a raw string literal is started but not
// closed, Read returns nil, ErrNoClosingRaw. If there is no data left to be
// read, Read returns nil, io.EOF.
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
	if r.Comment == r.Raw || r.Comment == r.Escape || r.Raw == r.Escape ||
		!validDelim(r.Comment) || !validDelim(r.Raw) || !validDelim(r.Escape) {
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
			if r.TrimLeadingSpace {
				line = strings.TrimLeftFunc(line, unicode.IsSpace)
			}

			// Skip empty lines or lines with only whitespace
			if line == "" {
				continue
			}

			// Check if the value is a raw string literal
			if c, size := utf8.DecodeRuneInString(line); c == r.Raw {
				inRaw = true
				line = line[size:]
			}
		}

		// Trim any comment not in raw string
		line = r.trimComment(line, inRaw)
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

				if r.isRaw(last, prev1, prev2) {
					rawString.WriteString(line[:j])
					line = rawString.String()
					rawString.Reset()
					inRaw = false
					break
				} else if last == r.Raw && prev1 == r.Escape {
					// Trim escape character
					line = line[:k] + line[j:]
				}
				rawString.WriteString(line)
			} else {
				line = strings.TrimRightFunc(line, unicode.IsSpace)
				line = strings.ReplaceAll(line, string(r.Escape)+string(r.Comment), string(r.Comment))
				if line == "" {
					continue
				}
				break
			}
		}

		if err != nil {
			break
		}
	}

	if inRaw {
		err = ErrNoClosingRaw
	}

	if err == nil {
		return line, nil
	}

	return "", err
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

// trimComment removes any comment that is not in a raw string literal.
func (r *Reader) trimComment(line string, inRaw bool) string {
	var prev1, prev2 rune
	for j, char := range line {
		if r.isComment(char, prev1, prev2) && !inRaw {
			line = line[:j]
			break
		} else if r.isRaw(char, prev1, prev2) && inRaw {
			inRaw = false
		}

		prev2 = prev1
		prev1 = char
	}

	return line
}

func (r *Reader) isComment(c, prev1, prev2 rune) bool {
	return isChar(r.Comment, r.Escape, c, prev1, prev2)
}

func (r *Reader) isRaw(c, prev1, prev2 rune) bool {
	return isChar(r.Raw, r.Escape, c, prev1, prev2)
}

// isChar determines if the rune at the index matches the char and that it is
// not escaped.
func isChar(char, escape, c, prev1, prev2 rune) bool {
	// Check if the character matches
	match := c == char

	// Check if the character is escaped
	isEsc1 := prev1 == escape

	// Check if the escape character is escaped
	isEsc2 := prev2 == escape

	return match && !isEsc1 && !(isEsc1 && isEsc2)
}
