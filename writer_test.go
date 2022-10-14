////////////////////////////////////////////////////////////////////////////////
// Copyright (c) 2022 jonow                                                   //
//                                                                            //
// Use of this source code is governed by an MIT-style license that can be    //
// found in the LICENSE file.                                                 //
////////////////////////////////////////////////////////////////////////////////

package lsv

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestNewWriter(t *testing.T) {
}

func TestWriter_Write(t *testing.T) {
}

func TestWriter_Flush(t *testing.T) {
}

func TestWriter_Error(t *testing.T) {
}

func TestWriter_WriteAll(t *testing.T) {
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
		t.Errorf("Output doesn't match input.\nexpected: %q\nreceived: %q", input, values)
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
		if val.value != values[i] {
			t.Errorf("Values #%d does not match.\nexpected: %q\nreceived: %q",
				i, val.value, values[i])
		}
	}
}

func TestWriter_valueNeedsEscaping(t *testing.T) {
}

func Test_firstRune(t *testing.T) {
}

func Test_lastRune(t *testing.T) {
}
