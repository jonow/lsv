////////////////////////////////////////////////////////////////////////////////
// Copyright (c) 2022 jonow                                                   //
//                                                                            //
// Use of this source code is governed by an MIT-style license that can be    //
// found in the LICENSE file.                                                 //
////////////////////////////////////////////////////////////////////////////////

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
	// ErrNoClosingRaw is returned when a quoted value is not closed.
	ErrNoClosingRaw = errors.New("raw literal not closed")

	// ErrInvalidParams is returned when the Parameters cannot be verified
	ErrInvalidParams = errors.New("invalid parameters")
)

// Reader reads values from a LSV-encoded file.
//
// The Reader expected input conforming to the LSV structure described in the
// README.md. The exported fields can be changed to customize the details before
// the first call to [Reader.Read] or [Reader.ReadAll].
type Reader struct {
	Parameters

	r *bufio.Reader
}

// NewReader returns a new Reader that reads from r.
func NewReader(r io.Reader) *Reader {
	return &Reader{
		Parameters: DefaultParameters(),
		r:          bufio.NewReader(r),
	}
}

// NewCustomReader returns a new Reader that reads from r with custom LSV
// parameters.
func NewCustomReader(r io.Reader, p Parameters) *Reader {
	return &Reader{
		Parameters: p,
		r:          bufio.NewReader(r),
	}
}

// ReadAll reads all the remaining values from r. A successful call returns
// err == nil, not err == io.EOF. Because ReadAll is defined to read until EOF,
// it does not treat end of file as an error to be reported.
func (r *Reader) ReadAll() ([]string, error) {
	if !r.Verify() {
		return nil, ErrInvalidParams
	}

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
	if !r.Verify() {
		return "", ErrInvalidParams
	}
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

				if r.isRaw(last, prev1) {
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
				// Trim trailing whitespace
				line = strings.TrimRightFunc(line, unicode.IsSpace)

				// Replace escaped comments with comment character
				line = strings.ReplaceAll(
					line, string(r.Escape)+string(r.Comment), string(r.Comment))

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
