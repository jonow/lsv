////////////////////////////////////////////////////////////////////////////////
// Copyright (c) 2022 jonow                                                   //
//                                                                            //
// Use of this source code is governed by an MIT-style license that can be    //
// found in the LICENSE file.                                                 //
////////////////////////////////////////////////////////////////////////////////

package lsv

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Split splits the LSV string into all substrings and returns a slice of all
// the values.
func Split(s string) ([]string, error) {
	return SplitParams(s, DefaultParameters())
}

// SplitParams splits the LSV string into its values with the specified
// Parameters.
func SplitParams(s string, p Parameters) ([]string, error) {
	if !p.Verify() {
		return nil, ErrInvalidParams
	}

	var inRaw bool
	var values []string
	var rawString strings.Builder

	for _, line := range strings.SplitAfter(s, "\n") {

		if !inRaw {
			// Trim leading whitespace if not in raw string literal
			if p.TrimLeadingSpace {
				line = strings.TrimLeftFunc(line, unicode.IsSpace)
			}

			// Skip empty lines or lines with only whitespace
			if line == "" {
				continue
			}

			// Check if the value is a raw string literal
			if c, size := utf8.DecodeRuneInString(line); c == p.Raw {
				inRaw = true
				line = line[size:]
			}
		}

		// Trim any comment not in raw string
		line = p.trimComment(line, inRaw)
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

				if p.isRaw(last, prev1) {
					rawString.WriteString(line[:j])
					values = append(values, rawString.String())
					rawString.Reset()
					inRaw = false
					continue
				} else if last == p.Raw && prev1 == p.Escape {
					// Trim escape character
					line = line[:k] + line[j:]
				}
				rawString.WriteString(line)
			} else {
				// Trim trailing whitespace
				line = strings.TrimRightFunc(line, unicode.IsSpace)

				// Replace escaped comments with comment character
				line = strings.ReplaceAll(
					line, string(p.Escape)+string(p.Comment), string(p.Comment))

				if len(line) > 0 {
					values = append(values, line)
					continue
				}
			}
		}
	}

	if inRaw {
		return nil, ErrNoClosingRaw
	}

	return values, nil
}
