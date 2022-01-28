package lsv

import (
	"reflect"
	"testing"
)

func Test_splitter(t *testing.T) {
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
test15`

	output, err := splitter(input)
	if err != nil {
		t.Errorf("Splitter errored: %+v", err)
	}

	if !reflect.DeepEqual(expected, output) {
		t.Errorf("Does not match.\nexpected: %q\nreceived: %q", expected, output)
	}

	t.Logf("%q", output)
}

// func Benchmark_splitter(b *testing.B) {
// 	line := "this is a value   # This is a comment\n"
// 	var str strings.Builder
// 	for n := 0; n < b.N; n++ {
// 		b.StopTimer()
// 		str.WriteString(line)
// 		buff := bytes.NewBufferString("")
// 		buff.WriteString(line)
// 		b.StartTimer()
// 		data, _ := io.ReadAll(buff)
// 		_, err := splitter(string(data))
// 		if err != nil {
// 			b.Errorf("splitter error: %+v", err)
// 		}
// 	}
// }
//
// func Benchmark_ReadAll(b *testing.B) {
// 	line := "this is a value   # This is a comment\n"
// 	var str strings.Builder
// 	for n := 0; n < b.N; n++ {
// 		b.StopTimer()
// 		str.WriteString(line)
// 		strReader := strings.NewReader(str.String())
// 		r := NewReader(strReader)
// 		b.StartTimer()
// 		_, err := r.ReadAll()
// 		if err != nil {
// 			b.Errorf("splitter error: %+v", err)
// 		}
// 	}
// }
