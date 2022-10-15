////////////////////////////////////////////////////////////////////////////////
// Copyright (c) 2022 jonow                                                   //
//                                                                            //
// Use of this source code is governed by an MIT-style license that can be    //
// found in the LICENSE file.                                                 //
////////////////////////////////////////////////////////////////////////////////

package lsv

import (
	"unicode"
	"unicode/utf8"
)

// Default runes.
const (
	defaultComment = '#'
	defaultRaw     = '"'
	defaultEscape  = '\\'
)

// Parameters contains customizable parameters for reading LSV files.
//
// Comment, Raw, and Escape must be valid according to [Parameters.Verify].
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

// DefaultParameters returns LSV Parameters with their default values.
func DefaultParameters() Parameters {
	return Parameters{
		Comment:          defaultComment,
		Raw:              defaultRaw,
		Escape:           defaultEscape,
		TrimLeadingSpace: true,
	}
}

// Verify checks that the Comment, Raw, and Escape are all unique and valid
// delimiters. A valid delimiter is any valid UTF-8 non-whitespace character
// that is not equal to 0 or [utf8.RuneError].
func (p Parameters) Verify() bool {
	return !(p.Comment == p.Raw || p.Comment == p.Escape || p.Raw == p.Escape ||
		!validDelim(p.Comment) || !validDelim(p.Raw) || !validDelim(p.Escape))
}

// validDelim determines if the rune is a valid delimiter
func validDelim(r rune) bool {
	return r != 0 &&
		!unicode.IsSpace(r) &&
		utf8.ValidRune(r) &&
		r != utf8.RuneError
}

// trimComment removes any comment that is not in a raw string literal. It does
// not trim whitespace.
func (p Parameters) trimComment(line string, inRaw bool) string {
	var prev rune
	for j, char := range line {
		if p.isComment(char, prev) && !inRaw {
			line = line[:j]
			break
		} else if p.isRaw(char, prev) && inRaw {
			inRaw = false
		}

		prev = char
	}

	return line
}

// isComment determines if the rune is an unescaped comment character.
func (p Parameters) isComment(c, prev rune) bool {
	return isChar(p.Comment, c, prev, p.Escape)
}

// isRaw determines if the rune is an unescaped raw character.
func (p Parameters) isRaw(c, prev rune) bool {
	return isChar(p.Raw, c, prev, p.Escape)
}

// isChar determines if c matches char and if c is unescaped. Returns true if
// they match. Returns false if they do not match or if c is escaped.
//
// char is the rune that we are matching c against. prev is the rune that appear
// before c. escapeChar is the rune used to escape characters.
func isChar(char, c, prev, escapeChar rune) bool {
	// Check if the character matches
	match := c == char

	// Check if the character is escaped
	isEsc := prev == escapeChar

	return match && !isEsc
}
