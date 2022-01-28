package lsv

import (
	"bufio"
	"errors"
	"io"
	"strings"
	"unicode"
)

// TODO: Add support for empty entries

const (
	defaultComment = '#'
	defaultRaw     = '"'
	defaultEscape  = '\\'
)

var ErrNoClosingRaw = errors.New("raw literal not closed")

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

	r *bufio.Reader
}

// NewReader returns a new Reader that reads from r.
func NewReader(r io.Reader) *Reader {
	return &Reader{
		Comment: defaultComment,
		Raw:     defaultRaw,
		Escape:  defaultEscape,
		r:       bufio.NewReader(r),
	}
}

func (r *Reader) Read() (string, error) {
	return r.readValue()
}

func (r *Reader) readValue() (string, error) {
	// TODO: check for valid delimiters

	var inRaw bool
	var line string
	var rawString strings.Builder
	var err error

	for {
		line, err = r.r.ReadString('\n')

		if line == "" && err == io.EOF {
			break
		}

		if !inRaw {
			// Trim leading whitespace if not in raw string literal
			line = strings.TrimLeftFunc(line, unicode.IsSpace)

			// Skip empty lines or lines with only whitespace
			if line == "" {
				continue
			}

			// Check if the value is a raw string literal
			if line[0] == '"' {
				inRaw = true
				line = line[1:]
			}
		}

		line = r.trimComment(line, inRaw)
		if line != "" {
			if inRaw {
				i := strings.LastIndexFunc(line, func(r rune) bool {
					return !unicode.IsSpace(r)
				})
				if i > -1 {
					if r.isRaw(i, line) {
						_, err = rawString.WriteString(line[:i])
						if err != nil {
							return "", err
						}

						line = rawString.String()
						rawString.Reset()
						inRaw = false
						break
					} else if line[i] == '"' && (i > 0 && line[i-1] == '\\') {
						// Trim escape character
						line = line[:i-1] + line[i:]
					}
				}
				_, err = rawString.WriteString(line)
				if err != nil {
					return "", err
				}
			} else {
				line = strings.TrimRightFunc(line, unicode.IsSpace)
				line = strings.ReplaceAll(line, "\\#", "#")
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

	if line != "" && err == io.EOF {
		err = nil
	}

	return line, err
}

func (r *Reader) ReadAll() ([]string, error) {
	var values []string

	for {
		value, err := r.readValue()
		// fmt.Printf("value: %q\n", value)
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
	for j := range line {
		if r.isComment(j, line) && !inRaw {
			line = line[:j]
			break
		} else if r.isRaw(j, line) && inRaw {
			inRaw = false
		}
	}

	return line
}

func (r *Reader) isComment(i int, str string) bool {
	return isChar(r.Comment, r.Escape, i, []rune(str))
}

func (r *Reader) isRaw(i int, str string) bool {
	return isChar(r.Raw, r.Escape, i, []rune(str))
}

// isChar determines if the rune at the index matches the char and that it is
// not escaped.
func isChar(char, escape rune, i int, str []rune) bool {
	// Check if the character matches
	match := str[i] == char

	// Check if the character is escaped
	isEsc1 := i > 0 && str[i-1] == escape

	// Check if the escape character is escaped
	isEsc2 := i > 1 && str[i-2] == escape

	return match && !isEsc1 && !(isEsc1 && isEsc2)
}
