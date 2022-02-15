package lsv

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Parameters tracks optional LSV parameters.
type Parameters struct {
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
}

// Split splits the LSV string into all substrings and returns a slice of all
// the values.
func Split(s string) ([]string, error) {
	p := Parameters{
		Comment:          defaultComment,
		Raw:              defaultRaw,
		Escape:           defaultEscape,
		TrimLeadingSpace: false,
	}
	return SplitParams(s, p)
}

// SplitParams splits the LSV string into its values with the specified
// Parameters.
func SplitParams(s string, p Parameters) ([]string, error) {
	if p.Comment == p.Raw || p.Comment == p.Escape || p.Raw == p.Escape ||
		!validDelim(p.Comment) || !validDelim(p.Raw) || !validDelim(p.Escape) {
		return nil, errInvalidDelim
	}

	var inRaw bool
	var values []string
	var rawString strings.Builder
	var err error

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

				if p.isRaw(last, prev1, prev2) {
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
	} else if err != nil {
		return nil, err
	}

	return values, nil
}

// trimComment removes any comment that is not in a raw string literal.
func (p Parameters) trimComment(line string, inRaw bool) string {
	r := Reader{
		Comment:          p.Comment,
		Raw:              p.Raw,
		Escape:           p.Escape,
		TrimLeadingSpace: p.TrimLeadingSpace,
	}

	return r.trimComment(line, inRaw)
}

// isComment determines if the rune is an unescaped comment character.
func (p Parameters) isComment(c, prev1, prev2 rune) bool {
	return isChar(p.Comment, p.Escape, c, prev1, prev2)
}

// isRaw determines if the rune is an unescaped raw character.
func (p Parameters) isRaw(c, prev1, prev2 rune) bool {
	return isChar(p.Raw, p.Escape, c, prev1, prev2)
}
