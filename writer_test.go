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
	Name:   "SingeLineComment",
	Input:  nil,
	Output: "# Comment",
}, {
	Name:   "EscapedSingeLineComment",
	Input:  []string{"# Comment"},
	Output: "\\# Comment",
}, {
	Name:   "InlineComment",
	Input:  []string{"a"},
	Output: "a # Comment",
}, {
	Name:   "EscapedInlineComment",
	Input:  []string{`a # Comment`},
	Output: `a \# Comment  `,
}, {
	Name:   "NonEscapedInlineComment",
	Input:  []string{`a \# Comment`},
	Output: `a \\# Comment  `,
}, {
	Name:   "NonEscapedInlineComment2",
	Input:  []string{`a \\# Comment`},
	Output: `a \\\# Comment  `,
}, {
	Name:   "SimpleWithComments",
	Input:  []string{"a", "b", "c"},
	Output: "a # Comment 1\nb\t# Comment 2\nc\f# Comment 3\n",
}, {
	Name:   "SimpleWithEscapedComments",
	Input:  []string{"a", "b\t# Comment 2", "c\f# Comment 3"},
	Output: "a # Comment 1\nb\t\\# Comment 2\n\"c\f# Comment 3\"\n",
}, {
	Name:   "NoEOLTestComment",
	Input:  []string{"a", "b", "c"},
	Output: "a\nb\nc # Comment",
}, {
	Name:   "BlankLineWithComments",
	Input:  []string{"a", "b", "c", "d", "e", "f"},
	Output: "a\nb\nc\n# Comment\nd\ne\nf\n#Comment\n",
}, {
	Name:   "TrimSpaceWithComment",
	Input:  []string{"a", "b", "c"},
	Output: " a # Comment\n  b # Comment\n   c # Comment\n",
}, {
	Name:   "LeadingSpaceWithComment",
	Input:  []string{" a", "  b", "   c"},
	Output: " a# Comment\n  b# Comment\n   c# Comment\n",
	NoTrim: true,
}, {
	Name:   "LeadingSpaceWithEscapedComment",
	Input:  []string{" a", "  b# Comment", "   c"},
	Output: " a# Comment\n  b\\# Comment\n   c# Comment\n",
	NoTrim: true,
}, {
	Name:   "BinaryBlobFieldWithComment",
	Input:  []string{"x09A\xb4\x1c", "AKTau"},
	Output: "x09\x41\xb4\x1c # Comment\nAKTau #comment",
}, {
	Name:   "QuotedFieldMultipleLFWithComment",
	Input:  []string{"\n\n\n\n"},
	Output: "\"\n\n\n\n\" # Comment",
}, {
	Name:   "MultipleCRLF",
	Input:  nil,
	Output: "\r\n\r\n\r\n\r\n # Comment",
}, {
	// The implementation may read each line in several chunks if it does not
	// fit entirely in the read buffer, so we should test the code to handle
	// that condition.
	Name: "HugeLinesWithComments",
	Input: []string{
		strings.Repeat("@", 5000),
		strings.Repeat("*", 5000) + " # Not a comment",
	},
	Output: strings.Repeat("# Comment\n", 10000) + "\n" +
		strings.Repeat("@", 5000) + " # Comment \n" +
		strings.Repeat("*", 5000) + " \\# Not a comment",
}, {
	Name:   "CRLFWithComments",
	Input:  []string{"a", "b", "c", "d"},
	Output: "a# Comment 1\nb # Comment 2\r\nc# Comment 3\nd\r\n",
}, {
	Name: "RawTestWithComments",
	Input: []string{"field1", "aaa", "bb \\# Not a comment\nb", "ccc", "a,a",
		`b"bb`, "ccc", "zzz", "yyy # Not a Comment", "xxx"},
	Output: `field1
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
}, {
	Name:   "EmptyValueWithComment",
	Input:  []string{""},
	Output: `"" # Comment`,
}, {
	Name:   "EmptyValueNewLineWithComment",
	Input:  []string{""},
	Output: "\"\"# Comment\n",
}, {
	Name:  "EmptyValuesWithComments",
	Input: []string{"x", "y", "z", "x", "", "y", "", "z", ""},
	Output: `x
# Comment
y

z # Comment

x
""
y
  ""  # Comment
z
""`,
}, {
	Name:    "NonASCIIComment",
	Input:   []string{"a", "b,c", "d,e"},
	Output:  "a\nb,c\n \td,e\n€ comment\n",
	Comment: '€',
}, {
	// λ and θ start with the same byte.
	// This tests that the parser doesn't confuse such characters.
	Name:    "NonASCIICommaConfusion",
	Input:   []string{"ABθCD", "EFθGH"},
	Output:  "\"ABθCD\" λ comment\nEFθGH λ comment",
	Comment: 'λ',
}, {
	Name:    "NonASCIICommentConfusion",
	Input:   []string{"λ", "λ", "λ"},
	Output:  "λ\nλ\nθ\nλθa\n",
	Comment: 'θ',
}, {
	Input:  []string{"a #A", "\\ \\#B", "#a"},
	Name:   "EscapedCommentConfusion",
	Output: "a \\#A\n\\ \\\\#B\n#\n\\#a\n",
}, {
	Name:    "NonASCIIEscapedCommentConfusion",
	Input:   []string{"a θA", "λ λθB", "θa"},
	Output:  "a λθA\nλ λλθB\nθ\nλθa\n",
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

// Tests that NewWriter returns a pointer to a new Writer with the expected
// default values.
func TestNewWriter_Consistency(t *testing.T) {
	buff := bytes.NewBufferString("")

	expected := &Writer{
		Comment:              defaultComment,
		LeadingCommentSpace:  defaultLeadingCommentSpace,
		TrailingCommentSpace: defaultTrailingCommentSpace,
		Raw:                  defaultRaw,
		Escape:               defaultEscape,
		UseCRLF:              false,
		w:                    bufio.NewWriter(buff),
	}

	newWriter := NewWriter(buff)

	if !reflect.DeepEqual(expected, newWriter) {
		t.Errorf("NewWriter did not return the expected writer."+
			"\nexpected: %+v\nreceived: %+v", expected, newWriter)
	}
}

func TestWriter_Write(t *testing.T) {

}
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

func TestWriter_WriteAll2(t *testing.T) {
	input := []string{
		"test1",
		"test2",
		"test3 #not a comment",
		"test4 \\#not a comment",
		"#not a comment",
		"\\#not a comment",
		"\\\\#not a comment",
		"Multi\nline",
		"   test5   # this it not a comment  ",
		"   test6    this it not a comment  ",
		"   test7   \"\n# this it not a comment  ",
		"\"",
		"test8\"test8",
		"test9\"",
		"\"test10\"",
		"\"test11\\\"\ntest11",
		"\"test12\"\ntest12",
		"\"test13\\\\\"\ntest13",
		"test14\n\ntest14",
		"",
		"test15",
	}
	buff := bytes.NewBufferString("")
	w := NewWriter(buff)
	err := w.WriteAll(input)
	if err != nil {
		t.Errorf("WriteAll error: %+v", err)
	}

	output := buff.String()

	t.Logf("output =====\n%s\n=======", output)

	r := NewReader(strings.NewReader(output))

	values, err := r.ReadAll()
	if err != nil {
		t.Errorf("ReadAll error: %+v", err)
	}

	if !reflect.DeepEqual(input, values) {
		t.Errorf("Output doesn't match input."+
			"\nexpected: %q\nreceived: %q", input, values)
	}

	for i, val := range input {
		if val != values[i] {
			t.Errorf("Values #%d does not match.\nexpected: %q\nreceived: %q",
				i, val, values[i])
		}
	}
}

func TestWriter_WriteAllWithComments(t *testing.T) {
	input := []ValueComment{
		{"", "This is a line only comment"},
		{"test1", "This is my comment"},
		{"test2", ""},
		{"test3 #not a comment", ""},
		{"test4 \\#not a comment", ""},
		{"#not a comment", ""},
		{"\\#not a comment", "THIS IS A REAL COMMENT"},
		{"\\\\#not a comment", ""},
		{"Multi\nline", ""},
		{"   test5   # this it not a comment  ", ""},
		{"   test6    this it not a comment  ", ""},
		{"   test7   \"\n# this it not a comment  ", ""},
		{"\"", ""},
		{"test8\"test8", ""},
		{"test9\"", ""},
		{"\"test10\"", ""},
		{"\"test11\\\"\ntest11", ""},
		{"\"test12\"\ntest12", ""},
		{"\"test13\\\\\"\ntest13", ""},
	}
	buff := bytes.NewBufferString("")
	w := NewWriter(buff)
	err := w.WriteAllWithComments(input)
	if err != nil {
		t.Errorf("WriteAll error: %+v", err)
	}

	output := buff.String()

	t.Logf("output =====\n%s\n=======", output)

	r := NewReader(strings.NewReader(output))

	values, err := r.ReadAll()
	if err != nil {
		t.Errorf("ReadAll error: %+v", err)
	}

	for i, val := range input[1:] {
		if val.Value != values[i] {
			t.Errorf("Values #%d does not match.\nexpected: %q\nreceived: %q",
				i, val.Value, values[i])
		}
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
