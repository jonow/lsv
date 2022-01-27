package lsv

import (
	"reflect"
	"strings"
	"testing"
)

func TestNewReader(t *testing.T) {
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

test14"`

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
