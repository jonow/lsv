package lsv

import (
	"bufio"
	"reflect"
	"strings"
	"testing"
)

type readTest struct {
	Name   string
	Input  string
	Output []string
	Error  error

	// These fields are copied into the Reader
	Comment          rune
	Raw              rune
	Escape           rune
	TrimLeadingSpace bool // Set to true to invert default
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
	Name: "RFC4180test",
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
	Name:   "NoEOLTest",
	Input:  "a\nb\nc",
	Output: []string{"a", "b", "c"},
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
	Name:   "BlankLine",
	Input:  "a\nb\nc\n\nd\ne\nf\n\n",
	Output: []string{"a", "b", "c", "d", "e", "f"},
}, {
	Name:   "TrimSpace",
	Input:  " a\n  b\n   c\n",
	Output: []string{"a", "b", "c"},
}, {
	Name:             "LeadingSpace",
	Input:            " a\n  b\n   c\n",
	Output:           []string{" a", "  b", "   c"},
	TrimLeadingSpace: true,
}, {
	Name:   "FullLineComment",
	Input:  "#1,2,3\na\nb\nc\n#comment",
	Output: []string{"a", "b", "c"},
}, {
	Name:   "InlineComment",
	Input:  "a #1\nb #2\nc #3\n",
	Output: []string{"a", "b", "c"},
}, {
	Name:    "NoComment",
	Input:   "#123\na\nb\nc",
	Output:  []string{"#123", "a", "b", "c"},
	Comment: '0',
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
	Name: "EmptyValueTest",
	Input: `x
y
z
w
x
y
z

x
y


x







"x"
"y"
"z"
"w"
"x"
"y"
"z"
""
"x"
"y"
""
""
"x"
""
""
""
""
""
""
""
`,
	Output: []string{"x", "y", "z", "w", "x", "y", "z", "x", "y", "x", "x", "y",
		"z", "w", "x", "y", "z", "", "x", "y", "", "", "x", "", "", "", "", "",
		"", ""},
}, {
	Name:   "CRLFInQuotedField",
	Input:  "A\n\"Hello\r\nHi\"\nB\r\n",
	Output: []string{"A", "Hello\r\nHi", "B"},
}, {
	Name:   "BinaryBlobField",
	Input:  "x09\x41\xb4\x1c\naktau",
	Output: []string{"x09A\xb4\x1c", "aktau"},
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
	Name:    "NonASCIIComment",
	Input:   "a\nb,c\n \td,e\n€ comment\n",
	Output:  []string{"a", "b,c", "d,e"},
	Comment: '€',
}, {
	// λ and θ start with the same byte.
	// This tests that the parser doesn't confuse such characters.
	Name:    "NonASCIICommaConfusion",
	Input:   "\"abθcd\" λ comment\nefθgh λ comment",
	Output:  []string{"abθcd", "efθgh"},
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
}, /*{
	Name:   "QuotedFieldMultipleLF",
	Input:  "\"\n\n\n\n\"",
	Output: [][]string{{"\n\n\n\n"}},
}, {
	Name:  "MultipleCRLF",
	Input: "\r\n\r\n\r\n\r\n",
}, {
	// The implementation may read each line in several chunks if it doesn't fit entirely
	// in the read buffer, so we should test the code to handle that condition.
	Name:    "HugeLines",
	Input:   strings.Repeat("#ignore\n", 10000) + "" + strings.Repeat("@", 5000) + "," + strings.Repeat("*", 5000),
	Output:  [][]string{{strings.Repeat("@", 5000), strings.Repeat("*", 5000)}},
	Comment: '#',
}, {
	Name:   "QuoteWithTrailingCRLF",
	Input:  "\"foo∑\"bar\"\r\n",
	Errors: []error{&ParseError{Err: ErrQuote}},
}, {
	Name:       "LazyQuoteWithTrailingCRLF",
	Input:      "\"foo\"bar\"\r\n",
	Output:     [][]string{{`foo"bar`}},
	LazyQuotes: true,
}, {
	Name:   "DoubleQuoteWithTrailingCRLF",
	Input:  "\"foo\"\"bar\"\r\n",
	Output: [][]string{{`foo"bar`}},
}, {
	Name:   "EvenQuotes",
	Input:  `""""""""`,
	Output: [][]string{{`"""`}},
}, {
	Name:   "OddQuotes",
	Input:  `"""""""∑`,
	Errors: []error{&ParseError{Err: ErrQuote}},
}, {
	Name:       "LazyOddQuotes",
	Input:      `"""""""`,
	Output:     [][]string{{`"""`}},
	LazyQuotes: true,
}, {
	Name:   "BadComma1",
	Comma:  '\n',
	Errors: []error{errInvalidDelim},
}, {
	Name:   "BadComma2",
	Comma:  '\r',
	Errors: []error{errInvalidDelim},
}, {
	Name:   "BadComma3",
	Comma:  '"',
	Errors: []error{errInvalidDelim},
}, {
	Name:   "BadComma4",
	Comma:  utf8.RuneError,
	Errors: []error{errInvalidDelim},
}, {
	Name:    "BadComment1",
	Comment: '\n',
	Errors:  []error{errInvalidDelim},
}, {
	Name:    "BadComment2",
	Comment: '\r',
	Errors:  []error{errInvalidDelim},
}, {
	Name:    "BadComment3",
	Comment: utf8.RuneError,
	Errors:  []error{errInvalidDelim},
}, {
	Name:    "BadCommaComment",
	Comma:   'X',
	Comment: 'X',
	Errors:  []error{errInvalidDelim},
},*/
}

// Tests that NewReader returns a pointer to a new Reader with the expected
// default values.
func TestNewReader(t *testing.T) {
	stringReader := strings.NewReader("test")

	expected := &Reader{
		Comment:          defaultComment,
		Raw:              defaultRaw,
		Escape:           defaultEscape,
		TrimLeadingSpace: true,
		r:                bufio.NewReader(stringReader),
	}

	newReader := NewReader(stringReader)

	if !reflect.DeepEqual(expected, newReader) {
		t.Errorf("NewReader did not return the expected reader."+
			"\nexpected: %+v\nreceived: %+v", expected, newReader)
	}
}

// Tests that Reader.Read reads each line of the values and return nil, io.EOF
// at the end.
func TestReader_Read(t *testing.T) {
	newReader := func(tt readTest) *Reader {
		r := NewReader(strings.NewReader(tt.Input))

		if tt.Comment != 0 {
			r.Comment = tt.Comment
		}
		if tt.Raw != 0 {
			r.Comment = tt.Raw
		}
		if tt.Escape != 0 {
			r.Escape = tt.Escape
		}
		if tt.TrimLeadingSpace {
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
					t.Fatalf("ReadAll() error mismatch:\ngot  %v (%#v)\nwant %v (%#v)", err, err, tt.Error, tt.Error)
				}
				if out != nil {
					t.Fatalf("ReadAll() output:\ngot  %q\nwant nil", out)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected Readall() error: %v", err)
				}
				if !reflect.DeepEqual(out, tt.Output) {
					t.Fatalf("ReadAll() output:\ngot  %q\nwant %q", out, tt.Output)
				}
			}
		})
	}

}

func TestReader_ReadAll(t *testing.T) {
	expected := []string{
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
		"test16©Sæ",
	}

	input := `
test1
test2   # comment
test3 \#not a comment
test4 \\#not a comment
# Comment only
\#not a comment
\\#not a comment
\\\#not a comment
"Multi
line"  # comment
    "   test5   # this it not a comment  "  # this is a comment   
    "   test6    this it not a comment  "  # this is a comment   
    "   test7   \"
# this it not a comment  "  # this is a comment   
"""
test8"test8
test9"
""test10""
""test11\\"
test11"
""test12\"
test12"
""test13\\\"
test13"
"test14

test14"
""
test15
test16©Sæ#comment`

	r := NewReader(strings.NewReader(input))
	output, err := r.ReadAll()
	if err != nil {
		t.Errorf("Splitter errored: %+v", err)
	}

	if !reflect.DeepEqual(expected, output) {
		t.Errorf("Does not match.\nexpected: %q\nreceived: %q", expected, output)
	}

	t.Logf("%q", output)
}

func TestReader_isComment(t *testing.T) {
}

func TestReader_isRaw(t *testing.T) {
}

func Test_isChar(t *testing.T) {
}
