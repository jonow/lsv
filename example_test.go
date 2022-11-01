////////////////////////////////////////////////////////////////////////////////
// Copyright (c) 2022 jonow                                                   //
//                                                                            //
// Use of this source code is governed by an MIT-style license that can be    //
// found in the LICENSE file.                                                 //
////////////////////////////////////////////////////////////////////////////////

package lsv

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

// This example shows how the [lsv.Reader] can read in a list with comments and
// quoted whitespace.
func ExampleReader() {
	in := `bananas
eggs # large
milk
apples
"  green"
"  red"
`
	r := NewReader(strings.NewReader(in))

	for {
		value, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(value)
	}
	// Output:
	// bananas
	// eggs
	// milk
	// apples
	//   green
	//   red
}

// This example shows how the [Reader.ReadAll] can read in the entire list at
// once.
func ExampleReader_ReadAll() {
	in := `bananas
eggs # large
milk
apples
"  green"
"  red"
`
	r := NewReader(strings.NewReader(in))

	values, err := r.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print(values)
	// Output:
	// [bananas eggs milk apples   green   red]
}

// This example shows how the [lsv.Writer] can read in a list of values, some
// with extra whitespace, and return a valid LSV.
func ExampleWriter() {
	values := []string{"bananas", "eggs", "milk", "apples", "  green", "  red"}

	w := NewWriter(os.Stdout)

	for _, value := range values {
		if err := w.Write(value); err != nil {
			log.Fatalln("error writing value to LSV:", err)
		}
	}

	// Write any buffered data to the underlying writer (standard output).
	w.Flush()

	if err := w.Error(); err != nil {
		log.Fatal(err)
	}
	// Output:
	// bananas
	// eggs
	// milk
	// apples
	// "  green"
	// "  red"
}

// This example shows how the [Writer.WriteAll] can write all the values to an
// LSV at once.
func ExampleWriter_WriteAll() {
	values := []string{"bananas", "eggs", "milk", "apples", "  green", "  red"}

	w := NewWriter(os.Stdout)

	_ = w.WriteAll(values) // calls Flush internally

	if err := w.Error(); err != nil {
		log.Fatalln("error writing LSV:", err)
	}
	// Output:
	// bananas
	// eggs
	// milk
	// apples
	// "  green"
	// "  red"
}

// This example shows how the [Writer.WriteAllWithComments] can write all the
// values, including their comments, to an LSV at once.
func ExampleWriter_WriteAllWithComments() {
	values := []ValueComment{
		{"bananas", ""},
		{"eggs", "This is my comment"},
		{"milk", ""},
		{"apples", "A comment"},
		{"  green", "Another comment"},
		{"  red", "A third comment"},
	}

	w := NewWriter(os.Stdout)

	_ = w.WriteAllWithComments(values) // calls Flush internally

	if err := w.Error(); err != nil {
		log.Fatalln("error writing LSV:", err)
	}
	// Output:
	// bananas
	// eggs	# This is my comment
	// milk
	// apples	# A comment
	// "  green"	# Another comment
	// "  red"	# A third comment
}
