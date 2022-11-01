////////////////////////////////////////////////////////////////////////////////
// Copyright (c) 2022 jonow                                                   //
//                                                                            //
// Use of this source code is governed by an MIT-style license that can be    //
// found in the LICENSE file.                                                 //
////////////////////////////////////////////////////////////////////////////////

package lsv

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

type writeTest struct {
	Name   string
	Input  []string
	Output string
	Error  error

	// These fields are copied into the Writer
	Comment rune
	Raw     rune
	Escape  rune
	UseCRLF bool
	NoTrim  bool // Set to true to invert default
}

var writeTests = []writeTest{{
	Name:   "Empty",
	Input:  []string{},
	Output: "",
}, {
	Name:   "Simple",
	Input:  []string{"a", "b", "c"},
	Output: "a\nb\nc\n",
}, {
	Name:    "CRLF",
	Input:   []string{"a", "b", "c", "d"},
	Output:  "a\r\nb\r\nc\r\nd\r\n",
	UseCRLF: true,
}, {
	Name:   "BareCR",
	Input:  []string{"a", "b\rc", "d"},
	Output: "a\nb\rc\nd\n",
}, {
	Name:   "LeadingSpace",
	Input:  []string{" a", "  b", "   c"},
	Output: "\" a\"\n\"  b\"\n\"   c\"\n",
}, {
	Name:   "BinaryBlobField",
	Input:  []string{"x09A\xb4\x1c", "AKTau"},
	Output: "x09\x41\xb4\x1c\nAKTau\n",
}, {
	Name:   "QuotedFieldMultipleLF",
	Input:  []string{"\n\n\n\n"},
	Output: "\"\n\n\n\n\"\n",
}, {
	// The implementation may read each line in several chunks if it does not
	// fit entirely in the read buffer, so we should test the code to handle
	// that condition.
	Name:   "HugeLines",
	Input:  []string{strings.Repeat("@", 5000), strings.Repeat("*", 5000)},
	Output: strings.Repeat("@", 5000) + "\n" + strings.Repeat("*", 5000) + "\n",
}, {
	Name:   "CRLFInQuotedField",
	Input:  []string{"A", "Hello\r\nHi", "B"},
	Output: "A\n\"Hello\r\nHi\"\nB\n",
}, {
	Name:   "QuoteWithTrailingCRLF",
	Input:  []string{`"foo"`, "\"bar\r"},
	Output: "\"\"foo\"\"\n\"\"bar\r\"\n",
}, {
	Name:   "LazyQuoteWithTrailingCRLF",
	Input:  []string{`foo"bar`},
	Output: "foo\"bar\n",
}, {
	Name:   "DoubleQuoteWithTrailingCRLF",
	Input:  []string{`foo""bar`},
	Output: "foo\"\"bar\n",
}, {
	Name:   "FieldCRCR",
	Input:  []string{"field\r\rfield"},
	Output: "field\r\rfield\n",
}, {
	Name:   "EvenQuotes",
	Input:  []string{`""""""`},
	Output: "\"\"\"\"\"\"\"\"\n",
}, {
	Name: "RawTest",
	Input: []string{"field1", "aaa", "bb\nb", "ccc", "a,a", `b"bb`, "ccc",
		"zzz", "yyy", "xxx"},
	Output: `field1
aaa
"bb
b"
ccc
a,a
b"bb
ccc
zzz
yyy
xxx
`,
}, {
	Name:   "EmptyValue",
	Input:  []string{""},
	Output: "\"\"\n",
}, {
	Name:   "EmptyValueNewLine",
	Input:  []string{""},
	Output: "\"\"\n",
}, {
	Name:   "TrailingLeadingSpaces",
	Input:  []string{"a", " b ", "   c   "},
	Output: "a\n\" b \"\n\"   c   \"\n",
}, {
	Name:  "MultiLine",
	Input: []string{"two\nline", "one line", "three\nline\nfield"},
	Output: `"two
line"
one line
"three
line
field"
`,
}, {
	Name:   "Quotes",
	Input:  []string{`a "word"`, `1"2`, `a"`, `b`},
	Output: "a \"word\"\n1\"2\na\"\nb\n",
}, {
	Name:   "BareDoubleQuotes",
	Input:  []string{`a""b`, `c`},
	Output: "a\"\"b\nc\n",
}, {
	Name:  "EscapedTrailingQuoteWithTest",
	Input: []string{"a", "b\nc\"\nd", "e"},
	Output: `a
"b
c\"
d"
e
`,
}, {
	Name:  "EscapedTrailingQuote",
	Input: []string{"a", "\n\"\n", "b"},
	Output: `a
"
\"
"
b
`,
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

type writeCommentTest struct {
	Name   string
	Input  []ValueComment
	Output string
	Error  error

	// These fields are copied into the Writer
	Comment rune
	Raw     rune
	Escape  rune
	UseCRLF bool
	NoTrim  bool // Set to true to invert default
}

var writeCommentTests = []writeCommentTest{{
	Name:   "SingeLineComment",
	Input:  []ValueComment{{"", "Comment"}},
	Output: "# Comment\n",
}, {
	Name:   "EscapedSingeLineComment",
	Input:  []ValueComment{{"# Comment", ""}},
	Output: "\\# Comment\n",
}, {
	Name:   "InlineComment",
	Input:  []ValueComment{{"a", "Comment"}},
	Output: "a\t# Comment\n",
}, {
	Name:   "EscapedInlineComment",
	Input:  []ValueComment{{"a # Comment  ", ""}},
	Output: "\"a # Comment  \"\n",
}, {
	Name:   "NonEscapedInlineComment",
	Input:  []ValueComment{{`\# Comment`, ""}},
	Output: "\\\\# Comment\n",
}, {
	Name:   "NonEscapedInlineComment2",
	Input:  []ValueComment{{`\\# Comment`, ""}},
	Output: "\\\\\\# Comment\n",
}, {
	Name:   "SimpleWithComments",
	Input:  []ValueComment{{"a", "Comment 1"}, {"b", "Comment 2"}, {"c", "Comment 3"}},
	Output: "a\t# Comment 1\nb\t# Comment 2\nc\t# Comment 3\n",
}, {
	Name:   "SimpleWithEscapedComments",
	Input:  []ValueComment{{"a", "Comment 1"}, {"b\t# Comment 2", ""}, {"c", "Comment 3"}},
	Output: "a\t# Comment 1\nb\t\\# Comment 2\nc\t# Comment 3\n",
}, {
	Name: "BlankLineWithComments",
	Input: []ValueComment{
		{"a", ""}, {"b", ""}, {"c", ""}, {"", "Comment"}, {"d", ""},
		{"e", ""}, {"f", ""}, {"", "Comment"}},
	Output: "a\nb\nc\n# Comment\nd\ne\nf\n# Comment\n",
}, {
	Name: "LeadingSpaceWithComment",
	Input: []ValueComment{
		{" a", "Comment"}, {"  b", "Comment"}, {"   c", "Comment"}},
	Output: "\" a\"\t# Comment\n\"  b\"\t# Comment\n\"   c\"\t# Comment\n",
}, {
	Name: "LeadingSpaceWithEscapedComment",
	Input: []ValueComment{
		{" a", ""}, {"  b\\# Comment", ""}, {"   c", ""}},
	Output: "\" a\"\n\"  b\\# Comment\"\n\"   c\"\n",
	NoTrim: true,
}, {
	Name:   "BinaryBlobFieldWithComment",
	Input:  []ValueComment{{"x09A\xb4\x1c", "Comment"}, {"AKTau", "comment"}},
	Output: "x09\x41\xb4\x1c\t# Comment\nAKTau\t# comment\n",
}, {
	Name:   "QuotedFieldMultipleLFWithComment",
	Input:  []ValueComment{{"\n\n\n\n", "Comment"}},
	Output: "\"\n\n\n\n\"\t# Comment\n",
}, {
	Name:   "EmptyValue",
	Input:  []ValueComment{{"", ""}},
	Output: "\"\"\n",
}, {
	Name:    "NonASCIIComment",
	Input:   []ValueComment{{"a", ""}, {"b,c", ""}, {"d,e", "comment"}},
	Output:  "a\nb,c\nd,e\t€ comment\n",
	Comment: '€',
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

// Tests that NewWriter returns a pointer to a new Writer with the expected
// default values.
func TestNewWriter_Consistency(t *testing.T) {
	buff := bytes.NewBufferString("")

	expected := &Writer{
		Parameters:           DefaultParameters(),
		LeadingCommentSpace:  defaultLeadingCommentSpace,
		TrailingCommentSpace: defaultTrailingCommentSpace,
		UseCRLF:              false,
		w:                    bufio.NewWriter(buff),
	}

	newWriter := NewWriter(buff)

	if !reflect.DeepEqual(expected, newWriter) {
		t.Errorf("NewWriter did not return the expected writer."+
			"\nexpected: %+v\nreceived: %+v", expected, newWriter)
	}
}

// Tests that Writer.WriteAll returns the expected error or value for each test.
func TestWriter_WriteAll(t *testing.T) {
	newWriter := func(tt writeTest, buff io.Writer) *Writer {
		w := NewWriter(buff)

		if tt.Comment != 0 {
			w.Comment = tt.Comment
		}
		if tt.Raw != 0 {
			w.Raw = tt.Raw
		}
		if tt.Escape != 0 {
			w.Escape = tt.Escape
		}
		w.UseCRLF = tt.UseCRLF
		return w
	}

	for _, tt := range writeTests {
		t.Run(tt.Name, func(t *testing.T) {
			buff := bytes.NewBufferString("")
			w := newWriter(tt, buff)
			err := w.WriteAll(tt.Input)
			if tt.Error != nil {
				if !reflect.DeepEqual(err, tt.Error) {
					t.Fatalf("WriteAll error mismatch:"+
						"\nexpected: %v (%#v)\nreceived: %v (%#v)",
						tt.Error, tt.Error, err, err)
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected WriteAll error: %+v", err)
				}
				out := buff.String()
				if out != tt.Output {
					t.Fatalf("WriteAll unexpected output:"+
						"\nexpected: %q\nreceived: %q", tt.Output, out)
				}
			}
		})
	}
}

// Tests that Writer.Write returns the expected error or value for each test.
func TestWriter_Write(t *testing.T) {
	newWriter := func(tt writeTest, buff io.Writer) *Writer {
		w := NewWriter(buff)

		if tt.Comment != 0 {
			w.Comment = tt.Comment
		}
		if tt.Raw != 0 {
			w.Raw = tt.Raw
		}
		if tt.Escape != 0 {
			w.Escape = tt.Escape
		}
		w.UseCRLF = tt.UseCRLF
		return w
	}

	for _, tt := range writeTests {
		t.Run(tt.Name, func(t *testing.T) {
			buff := bytes.NewBufferString("")
			w := newWriter(tt, buff)

			if tt.Input == nil {
				tt.Input = []string{""}
			}
			for _, line := range tt.Input {
				err := w.Write(line)
				t.Log(err)
				if tt.Error != nil {
					if err != nil {
						if !reflect.DeepEqual(err, tt.Error) {
							t.Fatalf("Write error mismatch:"+
								"\nexpected: %v (%#v)\nreceived: %v (%#v)",
								tt.Error, tt.Error, err, err)
						} else if line != "" {
							t.Fatalf("Write unexpected output:"+
								"\nexpected: nil\nreceived: %q", line)
						} else {
							return
						}
					}
				} else {
					if err != nil {
						t.Fatalf("Unexpected Write error: %+v", err)
					}
				}
			}

			w.Flush()
			out := buff.String()
			if out != tt.Output {
				t.Fatalf("Write unexpected output:"+
					"\nexpected: %q\nreceived: %q", tt.Output, out)
			}
		})
	}
}

// Tests that Writer.WriteAllWithComments returns the expected error or value
// for each test.
func TestWriter_WriteAllWithComments(t *testing.T) {
	newWriter := func(tt writeCommentTest, buff io.Writer) *Writer {
		w := NewWriter(buff)

		if tt.Comment != 0 {
			w.Comment = tt.Comment
		}
		if tt.Raw != 0 {
			w.Raw = tt.Raw
		}
		if tt.Escape != 0 {
			w.Escape = tt.Escape
		}
		w.UseCRLF = tt.UseCRLF
		return w
	}

	for _, tt := range writeCommentTests {
		t.Run(tt.Name, func(t *testing.T) {
			buff := bytes.NewBufferString("")
			w := newWriter(tt, buff)
			err := w.WriteAllWithComments(tt.Input)
			if tt.Error != nil {
				if !reflect.DeepEqual(err, tt.Error) {
					t.Fatalf("WriteAll error mismatch:"+
						"\nexpected: %v (%#v)\nreceived: %v (%#v)",
						tt.Error, tt.Error, err, err)
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected WriteAll error: %+v", err)
				}
				out := buff.String()
				if out != tt.Output {
					t.Fatalf("WriteAll unexpected output:"+
						"\nexpected: %q\nreceived: %q", tt.Output, out)
				}
			}
		})
	}
}

// Tests that Writer.WriteComment returns the expected error or value for each
// test.
func TestWriter_WriteComment(t *testing.T) {
	newWriter := func(tt writeCommentTest, buff io.Writer) *Writer {
		w := NewWriter(buff)

		if tt.Comment != 0 {
			w.Comment = tt.Comment
		}
		if tt.Raw != 0 {
			w.Raw = tt.Raw
		}
		if tt.Escape != 0 {
			w.Escape = tt.Escape
		}
		w.UseCRLF = tt.UseCRLF
		return w
	}

	for _, tt := range writeCommentTests {
		t.Run(tt.Name, func(t *testing.T) {
			buff := bytes.NewBufferString("")
			w := newWriter(tt, buff)

			if tt.Input == nil {
				tt.Input = []ValueComment{{}}
			}
			for _, line := range tt.Input {
				err := w.WriteComment(line.Value, line.Comment)
				if tt.Error != nil {
					if err != nil {
						if !reflect.DeepEqual(err, tt.Error) {
							t.Fatalf("Write error mismatch:"+
								"\nexpected: %v (%#v)\nreceived: %v (%#v)",
								tt.Error, tt.Error, err, err)
						} else if line != (ValueComment{}) {
							t.Fatalf("Write unexpected output:"+
								"\nexpected: nil\nreceived: %q", line)
						} else {
							return
						}
					}
				} else {
					if err != nil {
						t.Fatalf("Unexpected Write error: %+v", err)
					}
				}
			}

			w.Flush()
			out := buff.String()
			if out != tt.Output {
				t.Fatalf("Write unexpected output:"+
					"\nexpected: %q\nreceived: %q", tt.Output, out)
			}
		})
	}
}

type errorWriter struct{}

func (e errorWriter) Write([]byte) (int, error) {
	return 0, errors.New("test")
}

// Tests that Writer.Error returns an error when the underlying writer returns
// an error.
func TestWriter_Error(t *testing.T) {
	b := &bytes.Buffer{}
	f := NewWriter(b)
	_ = f.Write("abc")
	f.Flush()
	err := f.Error()
	if err != nil {
		t.Errorf("Unexpected error: %s\n", err)
	}

	f = NewWriter(errorWriter{})
	_ = f.Write("abc")
	f.Flush()
	err = f.Error()
	if err == nil {
		t.Error("Error should not be nil")
	}
}

// Tests that Writer.valueNeedsEscaping returns the expected value for each
// value in the test.
func TestWriter_valueNeedsEscaping(t *testing.T) {
	type test struct {
		Name   string
		W      *Writer
		Input  string
		Output bool
	}

	tests := []test{{
		"NormalString",
		NewWriter(nil),
		"This is a value",
		false,
	}, {
		"EmptyString",
		NewWriter(nil),
		"",
		false,
	}, {
		"LeadingSpace",
		NewWriter(nil),
		"  This is a value",
		true,
	}, {
		"TrailingSpace",
		NewWriter(nil),
		"This is a value  ",
		true,
	}, {
		"LiteralValue",
		NewWriter(nil),
		`"  This is the value  "`,
		true,
	}, {
		"NewLine",
		NewWriter(nil),
		"This\nis a value",
		true,
	},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			result := tt.W.valueNeedsEscaping(tt.Input)
			if tt.Output && !result {
				t.Fatalf("Value requires escaping: %q", tt.Input)
			} else if !tt.Output && result {
				t.Fatalf("Value does not require escaping: %q", tt.Input)
			}
		})
	}
}

// Tests that firstRune returns the expected character for each test.
func Test_firstRune(t *testing.T) {
	type test struct {
		Name   string
		Input  string
		Output rune
	}

	tests := []test{
		{"NormalString", "This is a normal string", 'T'},
		{"ShortString", "A", 'A'},
		{"EmptyString", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			result := firstRune(tt.Input)
			if result != tt.Output {
				t.Fatalf("Unexpected last rune.\nexpected: %c\nreceived: %c",
					tt.Output, result)
			}
		})
	}
}

// Tests that lastRune returns the expected character for each test.
func Test_lastRune(t *testing.T) {
	type test struct {
		Name   string
		Input  string
		Output rune
	}

	tests := []test{
		{"NormalString", "This is a normal string", 'g'},
		{"ShortString", "A", 'A'},
		{"EmptyString", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			result := lastRune(tt.Input)
			if result != tt.Output {
				t.Fatalf("Unexpected last rune.\nexpected: %c\nreceived: %c",
					tt.Output, result)
			}
		})
	}
}
