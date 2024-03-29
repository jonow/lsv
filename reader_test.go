////////////////////////////////////////////////////////////////////////////////
// Copyright (c) 2022 jonow                                                   //
//                                                                            //
// Use of this source code is governed by an MIT-style license that can be    //
// found in the LICENSE file.                                                 //
////////////////////////////////////////////////////////////////////////////////

package lsv

import (
	"bufio"
	"io"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

type readTest struct {
	Name   string
	Input  string
	Output []string
	Error  error

	// These fields are copied into the Reader
	Comment rune
	Raw     rune
	Escape  rune
	NoTrim  bool // Set to true to invert default
}

var readTests = []readTest{{
	Name:   "Simple",
	Input:  "a\nb\nc\n",
	Output: []string{"a", "b", "c"},
}, {
	Name:   "CRLF",
	Input:  "a\nb\r\nc\nd\r\n",
	Output: []string{"a", "b", "c", "d"},
}, {
	Name:   "BareCR",
	Input:  "a\nb\rc\nd\r\n",
	Output: []string{"a", "b\rc", "d"},
}, {
	Name:   "NoEOLTest",
	Input:  "a\nb\nc",
	Output: []string{"a", "b", "c"},
}, {
	Name:   "BlankLine",
	Input:  "a\nb\nc\n\nd\ne\nf\n\n",
	Output: []string{"a", "b", "c", "d", "e", "f"},
}, {
	Name:   "TrimSpace",
	Input:  " a\n  b\n   c\n",
	Output: []string{"a", "b", "c"},
}, {
	Name:   "LeadingSpace",
	Input:  " a\n  b\n   c\n",
	Output: []string{" a", "  b", "   c"},
	NoTrim: true,
}, {
	Name:   "BinaryBlobField",
	Input:  "x09\x41\xb4\x1c\nAKTau",
	Output: []string{"x09A\xb4\x1c", "AKTau"},
}, {
	Name:   "QuotedFieldMultipleLF",
	Input:  "\"\n\n\n\n\"",
	Output: []string{"\n\n\n\n"},
}, {
	Name:  "MultipleCRLF",
	Input: "\r\n\r\n\r\n\r\n",
}, {
	// The implementation may read each line in several chunks if it does not
	// fit entirely in the read buffer, so we should test the code to handle
	// that condition.
	Name: "HugeLines",
	Input: strings.Repeat("#ignore\n", 10000) + "" +
		strings.Repeat("@", 5000) + "\n" + strings.Repeat("*", 5000),
	Output: []string{strings.Repeat("@", 5000), strings.Repeat("*", 5000)},
}, {
	Name:   "CRLFInQuotedField",
	Input:  "A\n\"Hello\r\nHi\"\nB\r\n",
	Output: []string{"A", "Hello\r\nHi", "B"},
}, {
	Name:  "QuoteWithTrailingCRLF",
	Input: "\"foo\"\n\"bar\r\n",
	Error: ErrNoClosingRaw,
}, {
	Name:   "LazyQuoteWithTrailingCRLF",
	Input:  "\"foo\"bar\"\r\n",
	Output: []string{`foo"bar`},
}, {
	Name:   "DoubleQuoteWithTrailingCRLF",
	Input:  "\"foo\"\"bar\"\r\n",
	Output: []string{`foo""bar`},
}, {
	Name:   "TrailingCR",
	Input:  "field1\nfield2\r",
	Output: []string{"field1", "field2"},
}, {
	Name:   "QuotedTrailingCR",
	Input:  "\"field\"\r",
	Output: []string{"field"},
}, {
	Name:   "QuotedTrailingCRCR",
	Input:  "\"field\"\r\r",
	Output: []string{"field"},
}, {
	Name:   "FieldCR",
	Input:  "field\rfield\r",
	Output: []string{"field\rfield"},
}, {
	Name:   "FieldCRCR",
	Input:  "field\r\rfield\r\r",
	Output: []string{"field\r\rfield"},
}, {
	Name:   "EvenQuotes",
	Input:  `""""""""`,
	Output: []string{`""""""`},
}, {
	Name: "RawTest",
	Input: `field1
"aaa"
"bb
b"
"ccc"
"a,a"
"b"bb"
"ccc"
zzz
yyy
xxx
`,
	Output: []string{"field1", "aaa", "bb\nb", "ccc", "a,a", `b"bb`, "ccc",
		"zzz", "yyy", "xxx"},
}, {
	Name:   "EmptyValue",
	Input:  `""`,
	Output: []string{""},
}, {
	Name:   "EmptyValueNewLine",
	Input:  "\"\"\n",
	Output: []string{""},
}, {
	Name:   "TrailingLeadingSpaces",
	Input:  "a\n b \n   c   ",
	Output: []string{"a", "b", "c"},
}, {
	Name: "EmptyValues",
	Input: `x

y

z

x
""
y
  ""  
z
""`,
	Output: []string{"x", "y", "z", "x", "", "y", "", "z", ""},
}, {
	Name: "MultiLine",
	Input: `"two
line"
"one line"
"three
line
field"`,
	Output: []string{"two\nline", "one line", "three\nline\nfield"},
}, {
	Name:   "Quotes",
	Input:  "a \"word\"\n\"1\"2\"\na\"\n\"b\"",
	Output: []string{`a "word"`, `1"2`, `a"`, `b`},
}, {
	Name:   "BareQuotes",
	Input:  "a \"word\"\n\"1\"2\"\na\"",
	Output: []string{`a "word"`, `1"2`, `a"`},
}, {
	Name:   "BareDoubleQuotes",
	Input:  "a\"\"b\nc",
	Output: []string{`a""b`, `c`},
}, {
	Name: "TrimQuote",
	Input: ` "a"
" b"
c`,
	Output: []string{"a", " b", "c"},
}, {
	Name:  "ExtraneousQuote",
	Input: "a\n\"word\n\"b",
	Error: ErrNoClosingRaw,
}, {
	Name: "EscapedTrailingQuoteWithTest",
	Input: `a
"b
c\"
d"
e`,
	Output: []string{"a", "b\nc\"\nd", "e"},
}, {
	Name: "EscapedTrailingQuote",
	Input: `a
"
\"
"
b`,
	Output: []string{"a", "\n\"\n", "b"},
}, {
	Name:   "SingeLineComment",
	Input:  "# Comment",
	Output: nil,
}, {
	Name:   "EscapedSingeLineComment",
	Input:  "\\# Comment",
	Output: []string{"# Comment"},
}, {
	Name:   "InlineComment",
	Input:  "a # Comment",
	Output: []string{"a"},
}, {
	Name:   "EscapedInlineComment",
	Input:  `a \# Comment  `,
	Output: []string{`a # Comment`},
}, {
	Name:   "NonEscapedInlineComment",
	Input:  `a \\# Comment  `,
	Output: []string{`a \# Comment`},
}, {
	Name:   "NonEscapedInlineComment2",
	Input:  `a \\\# Comment  `,
	Output: []string{`a \\# Comment`},
}, {
	Name:   "SimpleWithComments",
	Input:  "a # Comment 1\nb\t# Comment 2\nc\f# Comment 3\n",
	Output: []string{"a", "b", "c"},
}, {
	Name:   "SimpleWithEscapedComments",
	Input:  "a # Comment 1\nb\t\\# Comment 2\n\"c\f# Comment 3\"\n",
	Output: []string{"a", "b\t# Comment 2", "c\f# Comment 3"},
}, {
	Name:   "NoEOLTestComment",
	Input:  "a\nb\nc # Comment",
	Output: []string{"a", "b", "c"},
}, {
	Name:   "BlankLineWithComments",
	Input:  "a\nb\nc\n# Comment\nd\ne\nf\n#Comment\n",
	Output: []string{"a", "b", "c", "d", "e", "f"},
}, {
	Name:   "TrimSpaceWithComment",
	Input:  " a # Comment\n  b # Comment\n   c # Comment\n",
	Output: []string{"a", "b", "c"},
}, {
	Name:   "LeadingSpaceWithComment",
	Input:  " a# Comment\n  b# Comment\n   c# Comment\n",
	Output: []string{" a", "  b", "   c"},
	NoTrim: true,
}, {
	Name:   "LeadingSpaceWithEscapedComment",
	Input:  " a# Comment\n  b\\# Comment\n   c# Comment\n",
	Output: []string{" a", "  b# Comment", "   c"},
	NoTrim: true,
}, {
	Name:   "BinaryBlobFieldWithComment",
	Input:  "x09\x41\xb4\x1c # Comment\nAKTau #comment",
	Output: []string{"x09A\xb4\x1c", "AKTau"},
}, {
	Name:   "QuotedFieldMultipleLFWithComment",
	Input:  "\"\n\n\n\n\" # Comment",
	Output: []string{"\n\n\n\n"},
}, {
	Name:   "MultipleCRLF",
	Input:  "\r\n\r\n\r\n\r\n # Comment",
	Output: nil,
}, {
	// The implementation may read each line in several chunks if it does not
	// fit entirely in the read buffer, so we should test the code to handle
	// that condition.
	Name: "HugeLinesWithComments",
	Input: strings.Repeat("# Comment\n", 10000) + "\n" +
		strings.Repeat("@", 5000) + " # Comment \n" +
		strings.Repeat("*", 5000) + " \\# Not a comment",
	Output: []string{
		strings.Repeat("@", 5000),
		strings.Repeat("*", 5000) + " # Not a comment",
	},
}, {
	Name:   "CRLFWithComments",
	Input:  "a# Comment 1\nb # Comment 2\r\nc# Comment 3\nd\r\n",
	Output: []string{"a", "b", "c", "d"},
}, {
	Name: "RawTestWithComments",
	Input: `field1
"aaa" # Comment 1
"bb \# Not a comment
b" # Comment 2
"ccc"
"a,a"
"b"bb" # Comment 3
"ccc"
zzz
yyy \# Not a Comment
xxx
`,
	Output: []string{"field1", "aaa", "bb \\# Not a comment\nb", "ccc", "a,a",
		`b"bb`, "ccc", "zzz", "yyy # Not a Comment", "xxx"},
}, {
	Name:   "EmptyValueWithComment",
	Input:  `"" # Comment`,
	Output: []string{""},
}, {
	Name:   "EmptyValueNewLineWithComment",
	Input:  "\"\"# Comment\n",
	Output: []string{""},
}, {
	Name: "EmptyValuesWithComments",
	Input: `x
# Comment
y

z # Comment

x
""
y
  ""  # Comment
z
""`,
	Output: []string{"x", "y", "z", "x", "", "y", "", "z", ""},
}, {
	Name:    "NonASCIIComment",
	Input:   "a\nb,c\n \td,e\n€ comment\n",
	Output:  []string{"a", "b,c", "d,e"},
	Comment: '€',
}, {
	// λ and θ start with the same byte.
	// This tests that the parser doesn't confuse such characters.
	Name:    "NonASCIICommaConfusion",
	Input:   "\"ABθCD\" λ comment\nEFθGH λ comment",
	Output:  []string{"ABθCD", "EFθGH"},
	Comment: 'λ',
}, {
	Name:    "NonASCIICommentConfusion",
	Input:   "λ\nλ\nθ\nλθa\n",
	Output:  []string{"λ", "λ", "λ"},
	Comment: 'θ',
}, {
	Name:   "EscapedCommentConfusion",
	Input:  "a \\#A\n\\ \\\\#B\n#\n\\#a\n",
	Output: []string{"a #A", "\\ \\#B", "#a"},
}, {
	Name:    "NonASCIIEscapedCommentConfusion",
	Input:   "a λθA\nλ λλθB\nθ\nλθa\n",
	Output:  []string{"a θA", "λ λθB", "θa"},
	Comment: 'θ',
	Escape:  'λ',
}, {
	Name:    "BadComment_IsSpace",
	Comment: ' ',
	Error:   ErrInvalidParams,
}, {
	Name:    "BadComment_InvalidRune",
	Comment: 0xD800,
	Error:   ErrInvalidParams,
}, {
	Name:    "BadComment_utf8RuneError",
	Comment: utf8.RuneError,
	Error:   ErrInvalidParams,
}, {
	Name:    "BadComment_SameAsRaw",
	Comment: '"',
	Error:   ErrInvalidParams,
}, {
	Name:    "BadComment_SameAsEscape",
	Comment: '\\',
	Error:   ErrInvalidParams,
}, {
	Name:  "BadRaw_IsSpace",
	Raw:   '\r',
	Error: ErrInvalidParams,
}, {
	Name:  "BadRaw_InvalidRune",
	Raw:   0xDFFF,
	Error: ErrInvalidParams,
}, {
	Name:  "BadRaw_utf8RuneError",
	Raw:   utf8.RuneError,
	Error: ErrInvalidParams,
}, {
	Name:    "BadRaw_SameAsComment",
	Comment: '#',
	Raw:     '#',
	Error:   ErrInvalidParams,
}, {
	Name:   "BadRaw_SameAsEscape",
	Raw:    '\\',
	Escape: '\\',
	Error:  ErrInvalidParams,
}, {
	Name:   "BadEscape_IsSpace",
	Escape: '\n',
	Error:  ErrInvalidParams,
}, {
	Name:   "BadEscape_InvalidRune",
	Escape: -1,
	Error:  ErrInvalidParams,
}, {
	Name:   "BadEscape_utf8RuneError",
	Escape: utf8.RuneError,
	Error:  ErrInvalidParams,
}, {
	Name:    "BadEscape_SameAsComment",
	Comment: '#',
	Escape:  '#',
	Error:   ErrInvalidParams,
}, {
	Name:   "BadEscape_SameAsRaw",
	Raw:    '"',
	Escape: '"',
	Error:  ErrInvalidParams,
},
}

// Tests that NewReader returns a pointer to a new Reader with the expected
// default values.
func TestNewReader_Consistency(t *testing.T) {
	stringReader := strings.NewReader("test")

	expected := &Reader{
		Parameters: DefaultParameters(),
		r:          bufio.NewReader(stringReader),
	}

	newReader := NewReader(stringReader)

	if !reflect.DeepEqual(expected, newReader) {
		t.Errorf("NewReader did not return the expected reader."+
			"\nexpected: %+v\nreceived: %+v", expected, newReader)
	}
}

// Tests that NewCustomReader returns a pointer to a new Reader with the
// expected values.
func TestNewCustomReader(t *testing.T) {
	stringReader := strings.NewReader("test")

	expected := &Reader{
		Parameters: Parameters{
			Comment:          'C',
			Raw:              '&',
			Escape:           'E',
			TrimLeadingSpace: false,
		},
		r: bufio.NewReader(stringReader),
	}

	newReader := NewCustomReader(stringReader, expected.Parameters)

	if !reflect.DeepEqual(expected, newReader) {
		t.Errorf("NewCustomReader did not return the expected reader."+
			"\nexpected: %+v\nreceived: %+v", expected, newReader)
	}
}

// Tests that Reader.ReadAll returns the expected error or value for each test.
func TestReader_ReadAll(t *testing.T) {
	newReader := func(tt readTest) *Reader {
		r := NewReader(strings.NewReader(tt.Input))

		if tt.Comment != 0 {
			r.Comment = tt.Comment
		}
		if tt.Raw != 0 {
			r.Raw = tt.Raw
		}
		if tt.Escape != 0 {
			r.Escape = tt.Escape
		}
		if tt.NoTrim {
			r.TrimLeadingSpace = false
		}
		return r
	}

	for _, tt := range readTests {
		t.Run(tt.Name, func(t *testing.T) {
			r := newReader(tt)
			out, err := r.ReadAll()
			if tt.Error != nil {
				if !reflect.DeepEqual(err, tt.Error) {
					t.Fatalf("ReadAll error mismatch:"+
						"\nexpected: %v (%#v)\nreceived: %v (%#v)",
						tt.Error, tt.Error, err, err)
				}
				if out != nil {
					t.Fatalf("ReadAll unexpected output:"+
						"\nexpected: nil\nreceived: %q", out)
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected Readall error: %+v", err)
				}
				if !reflect.DeepEqual(out, tt.Output) {
					t.Fatalf("ReadAll unexpected output:"+
						"\nexpected: %q\nreceived: %q", tt.Output, out)
				}
			}
		})
	}
}

// Tests that Reader.Read returns the expected error or value for each test.
func TestReader_Read(t *testing.T) {
	newReader := func(tt readTest) *Reader {
		r := NewReader(strings.NewReader(tt.Input))

		if tt.Comment != 0 {
			r.Comment = tt.Comment
		}
		if tt.Raw != 0 {
			r.Raw = tt.Raw
		}
		if tt.Escape != 0 {
			r.Escape = tt.Escape
		}
		if tt.NoTrim {
			r.TrimLeadingSpace = false
		}
		return r
	}

	for _, tt := range readTests {
		t.Run(tt.Name, func(t *testing.T) {
			r := newReader(tt)

			i := 0
			for line, err := r.Read(); err != io.EOF; line, err = r.Read() {
				if tt.Error != nil {
					if err != nil {
						if !reflect.DeepEqual(err, tt.Error) {
							t.Fatalf("Read error mismatch:"+
								"\nexpected: %v (%#v)\nreceived: %v (%#v)",
								tt.Error, tt.Error, err, err)
						} else if line != "" {
							t.Fatalf("Read unexpected output:"+
								"\nexpected: nil\nreceived: %q", line)
						} else {
							return
						}
					}
				} else {
					if err != nil {
						t.Fatalf("Unexpected Read error: %+v", err)
					}
					if line != tt.Output[i] {
						t.Fatalf("ReadAll unexpected output:"+
							"\nexpected: %q\nreceived: %q", tt.Output[i], line)
					}
				}
				i++
			}

			if tt.Error != nil {
				t.Fatalf("Read failed to error. Expected error: %v", tt.Error)
			}
		})
	}
}
