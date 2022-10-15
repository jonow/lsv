////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package lsv

import (
	"reflect"
	"testing"
	"unicode/utf8"
)

// Consistency test of DefaultParameters.
func TestDefaultParameters_Consistency(t *testing.T) {
	expected := Parameters{
		Comment:          defaultComment,
		Raw:              defaultRaw,
		Escape:           defaultEscape,
		TrimLeadingSpace: true,
	}

	p := DefaultParameters()

	if !reflect.DeepEqual(expected, p) {
		t.Errorf("Default Parameters do not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, p)
	}
}

// Tests that Parameters.Verify returns the expected output for various valid
// and Parameters.
func TestParameters_Verify(t *testing.T) {
	type test struct {
		Name   string
		P      Parameters
		Output bool
	}

	tests := []test{{
		"ValidDefault",
		DefaultParameters(),
		true,
	}, {
		"Valid",
		Parameters{
			Comment: 'A',
			Raw:     'B',
			Escape:  'C',
		},
		true,
	}, {
		"InvalidMatchCommentAndRaw",
		Parameters{
			Comment: 'A',
			Raw:     'A',
			Escape:  'C',
		},
		false,
	}, {
		"InvalidMatchCommentAndEscape",
		Parameters{
			Comment: 'A',
			Raw:     'B',
			Escape:  'A',
		},
		false,
	}, {
		"InvalidMatchRawAndEscape",
		Parameters{
			Comment: 'A',
			Raw:     'B',
			Escape:  'B',
		},
		false,
	}, {
		"InvalidMatchCommentRawAndEscape",
		Parameters{
			Comment: 'A',
			Raw:     'A',
			Escape:  'A',
		},
		false,
	}, {
		"InvalidCommentDelim",
		Parameters{
			Comment: ' ',
			Raw:     'A',
			Escape:  'B',
		},
		false,
	}, {
		"InvalidRawDelim",
		Parameters{
			Comment: 'A',
			Raw:     '\n',
			Escape:  'B',
		},
		false,
	}, {
		"InvalidEscapeDelim",
		Parameters{
			Comment: 'A',
			Raw:     'B',
			Escape:  0,
		},
		false,
	},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			result := tt.P.Verify()
			if tt.Output && !result {
				t.Fatalf("Valid parameters marked invalid: %+v.", tt.P)
			} else if !tt.Output && result {
				t.Fatalf("Invalid parameters marked valid: %+v.", tt.P)
			}
		})
	}
}

// Tests that validDelim returns the expected output for various valid and
// invalid delimiters.
func Test_validDelim(t *testing.T) {
	type test struct {
		Name   string
		R      rune
		Output bool
	}

	tests := []test{
		{"Letter", 'A', true},
		{"Number", '0', true},
		{"Symbol", '?', true},
		{"Zero", 0, false},
		{"Space", ' ', false},
		{"Tab", '\t', false},
		{"NewLine", '\n', false},
		{"CarriageReturn", '\r', false},
		{"InvalidRune", 0xD800, false},
		{"RuneError", utf8.RuneError, false},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			result := validDelim(tt.R)
			if tt.Output && !result {
				t.Fatalf("Valid delimiter %c marked as invalid.", tt.R)
			} else if !tt.Output && result {
				t.Fatalf("Invalid delimiter %c marked as valid.", tt.R)
			}
		})
	}
}

// Tests that Parameters.trimComment returns the expected output for each test.
func TestParameters_trimComment(t *testing.T) {
	type test struct {
		Name   string
		P      Parameters
		Line   string
		InRaw  bool
		Output string
	}

	tests := []test{{
		"NoComment",
		DefaultParameters(),
		"This is a normal line of text.",
		false,
		"This is a normal line of text.",
	}, {
		"NormalCommentWhitespace",
		DefaultParameters(),
		"This is a normal line of text. # My comment",
		false,
		"This is a normal line of text. ",
	}, {
		"NormalCommentNoWhitespace",
		DefaultParameters(),
		"This is a normal line of text.# My comment",
		false,
		"This is a normal line of text.",
	}, {
		"EscapedComment",
		DefaultParameters(),
		`This is a normal line of text. \# My comment`,
		false,
		`This is a normal line of text. \# My comment`,
	}, {
		"FullStringLiteral",
		DefaultParameters(),
		`"This is a normal line of text."`,
		true,
		`"This is a normal line of text."`,
	}, {
		"FullStringLiteralWithCommentCharacter",
		DefaultParameters(),
		`This is a normal line of text. # Not a comment"`,
		true,
		`This is a normal line of text. # Not a comment"`,
	}, {
		"FullStringLiteralWithComment",
		DefaultParameters(),
		`"This is a normal line of text." # Not a comment`,
		true,
		`"This is a normal line of text." `,
	}, {
		"FullStringLiteralWithEscapedComment",
		DefaultParameters(),
		`"This is a normal line of text." \# Not a comment`,
		true,
		`"This is a normal line of text." \# Not a comment`,
	},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			result := tt.P.trimComment(tt.Line, tt.InRaw)
			if result != tt.Output {
				t.Fatalf("Line was not properly trimmed of comments."+
					"\nexpected: %q\nreceived: %q", tt.Output, result)
			}
		})
	}
}

// Tests that Parameters.isComment returns the expected output for each test.
func TestParameters_isComment(t *testing.T) {
	type test struct {
		Name    string
		P       Parameters
		C, Prev rune
		Output  bool
	}

	tests := []test{{
		"NormalMatch",
		DefaultParameters(),
		defaultComment, 'B',
		true,
	}, {
		"EscapeNoMatch",
		DefaultParameters(),
		defaultComment, defaultEscape,
		false,
	}, {
		"NoMatch",
		DefaultParameters(),
		'A', 'B',
		false,
	},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			result := tt.P.isComment(tt.C, tt.Prev)
			if tt.Output && !result {
				t.Fatalf(
					"Char not recognize as raw char: %c == %c", tt.C, tt.P.Raw)
			} else if !tt.Output && result {
				t.Fatalf("Char recognize as raw char when it should not have: "+
					"%c != %c", tt.C, tt.P.Raw)
			}
		})
	}
}

// Tests that Parameters.isRaw returns the expected output for each test.
func TestParameters_isRaw(t *testing.T) {
	type test struct {
		Name    string
		P       Parameters
		C, Prev rune
		Output  bool
	}

	tests := []test{{
		"NormalMatch",
		DefaultParameters(),
		defaultRaw, 'B',
		true,
	}, {
		"EscapeNoMatch",
		DefaultParameters(),
		defaultRaw, defaultEscape,
		false,
	}, {
		"NoMatch",
		DefaultParameters(),
		'A', 'B',
		false,
	},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			result := tt.P.isRaw(tt.C, tt.Prev)
			if tt.Output && !result {
				t.Fatalf(
					"Char not recognize as raw char: %c == %c", tt.C, tt.P.Raw)
			} else if !tt.Output && result {
				t.Fatalf("Char recognize as raw char when it should not have: "+
					"%c != %c", tt.C, tt.P.Raw)
			}
		})
	}
}

// Tests that isChar returns the expected output for each test.
func Test_isChar(t *testing.T) {
	type test struct {
		Name                      string
		Char, C, Prev, EscapeChar rune
		Output                    bool
	}

	tests := []test{{
		"NormalMatch",
		'B', 'B', 'A', defaultEscape,
		true,
	}, {
		"EscapedNoMatch",
		'#', '#', defaultEscape, defaultEscape,
		false,
	}, {
		"EscapedEscapeChar",
		defaultEscape, defaultEscape, defaultEscape, defaultEscape,
		false,
	}, {
		"EscapeChar",
		defaultEscape, defaultEscape, 'A', defaultEscape,
		true,
	},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			result := isChar(tt.Char, tt.C, tt.Prev, tt.EscapeChar)
			if tt.Output && !result {
				t.Fatalf("Two equal chars determined not equal: %c == %c",
					tt.Char, tt.C)
			} else if !tt.Output && result {
				t.Fatalf("Two unequal chars determined to be equal. %c != %c",
					tt.Char, tt.C)
			}
		})
	}
}
