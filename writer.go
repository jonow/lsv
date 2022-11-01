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
	"strings"
	"unicode"
)

const (
	defaultLeadingCommentSpace  = "\t"
	defaultTrailingCommentSpace = " "
)

// Writer writes values using LSV encoding.
//
// The Writer writes records according to the LSV structure described in the
// README.md. The exported fields can be changed to customize the details before
// the first call to [Writer.Write] or [Writer.WriteAll].
//
// If UseCRLF is true, the Writer ends each output line with \r\n instead of \n.
//
// The writes of individual values are buffered. After all data has been
// written, the user should call the [Writer.Flush] method to guarantee all data
// has been forwarded to the underlying [io.Writer]. Any errors that occurred
// should be checked by calling the [Writer.Error] method.
type Writer struct {
	Parameters

	// LeadingCommentSpace is the space written before the Comment character
	// when writing a comment.
	LeadingCommentSpace string

	// TrailingCommentSpace is the space written after the Comment character
	// when writing a comment.
	TrailingCommentSpace string

	// UseCRLF uses \r\n as the line terminator if set to true.
	UseCRLF bool

	w *bufio.Writer
}

// NewWriter returns a new Writer that write to w.
func NewWriter(w io.Writer) *Writer {
	return &Writer{
		Parameters:           DefaultParameters(),
		LeadingCommentSpace:  defaultLeadingCommentSpace,
		TrailingCommentSpace: defaultTrailingCommentSpace,
		UseCRLF:              false,
		w:                    bufio.NewWriter(w),
	}
}

// WriteAll writes multiple LSV records to w using [Writer.Write] and then calls
// [Writer.Flush], returning any error from the [Writer.Flush].
func (w *Writer) WriteAll(values []string) error {
	if !w.Verify() {
		return ErrInvalidParams
	}

	for _, value := range values {
		err := w.writeComment(value, "")
		if err != nil {
			return err
		}
	}
	return w.w.Flush()
}

// Write writes a single LSV value to w along with any necessary quoting and
// escaping.
//
// Writes are buffered, so [Writer.Flush] must eventually be called to ensure
// that the record is written to the underlying [io.Writer].
func (w *Writer) Write(value string) error {
	if !w.Verify() {
		return ErrInvalidParams
	}
	return w.writeComment(value, "")
}

// ValueComment represents a single value in an LSV file. This structure
// supports including a comment on a line when writing to an LSV.
type ValueComment struct {
	Value, Comment string
}

// WriteAllWithComments writes multiple LSV records and their optional comments
// to w using [Writer.Write] and then calls [Writer.Flush], returning any error
// from the [Writer.Flush].
func (w *Writer) WriteAllWithComments(values []ValueComment) error {
	if !w.Verify() {
		return ErrInvalidParams
	}

	for _, value := range values {
		err := w.writeComment(value.Value, value.Comment)
		if err != nil {
			return err
		}
	}
	return w.w.Flush()
}

// WriteComment writes a single LSV record to w along with any necessary quoting
// and escaping. If a comment is included, then it is appended to the end of the
// value.
//
// Writes are buffered, so [Writer.Flush] must eventually be called to ensure
// that the record is written to the underlying [io.Writer].
func (w *Writer) WriteComment(value, comment string) error {
	if !w.Verify() {
		return ErrInvalidParams
	}
	return w.writeComment(value, comment)
}

// writeComment writes a single LSV record to w with an included comment,
// if specified.
func (w *Writer) writeComment(value, comment string) error {
	var bytesWritten, n int
	var err error

	// If the value does not need to be escaped, then write the value to the
	// buffer
	if value != "" {
		if !w.valueNeedsEscaping(value) {
			n, err = strings.NewReplacer(
				"\\#", "\\\\#", "#", "\\#").WriteString(w.w, value)
			if err != nil {
				return err
			}

			bytesWritten += n
		} else {
			n, err = w.w.WriteRune('"')
			if err != nil {
				return err
			}
			bytesWritten += n

			n, err = strings.NewReplacer(
				"\\\"\n", "\\\\\"\n", "\"\n", "\\\"\n").WriteString(w.w, value)
			if err != nil {
				return err
			}
			bytesWritten += n

			n, err = w.w.WriteRune('"')
			if err != nil {
				return err
			}
			bytesWritten += n
		}
	}

	if comment != "" {
		if bytesWritten > 0 {
			n, err = w.w.WriteString(w.LeadingCommentSpace)
			if err != nil {
				return err
			}
			bytesWritten += n
		}
		n, err = w.w.WriteRune(w.Comment)
		if err != nil {
			return err
		}
		bytesWritten += n

		n, err = w.w.WriteString(w.TrailingCommentSpace)
		if err != nil {
			return err
		}
		bytesWritten += n

		n, err = w.w.WriteString(comment)
		if err != nil {
			return err
		}
		bytesWritten += n
	}

	if value == "" && comment == "" {
		n, err = w.w.WriteRune(w.Raw)
		if err != nil {
			return err
		}
		bytesWritten += n
		n, err = w.w.WriteRune(w.Raw)
		if err != nil {
			return err
		}
		bytesWritten += n
	}

	// Do not add delimiter if no value was written
	if bytesWritten > 0 {
		if w.UseCRLF {
			_, err = w.w.WriteString("\r\n")
		} else {
			err = w.w.WriteByte('\n')
		}
	}
	return err
}

// Flush writes any buffered data to the underlying [io.Writer]. To check if an
// error occurred during the [Writer.Flush], call [Writer.Error].
func (w *Writer) Flush() {
	_ = w.w.Flush()
}

// Error reports any error that has occurred during a previous [Writer.Write] or
// [Writer.Flush].
func (w *Writer) Error() error {
	_, err := w.w.Write(nil)
	return err
}

// valueNeedsEscaping determines if the value needs to be escaped. Values with
// leading/trailing whitespace, newlines, or quotes at end of lines need to be
// escaped.
func (w *Writer) valueNeedsEscaping(value string) bool {
	if value == "" {
		return false
	}

	// Check for leading and/or trailing whitespace
	if unicode.IsSpace(firstRune(value)) || unicode.IsSpace(lastRune(value)) {
		return true
	}

	// Check for leading raw character
	if firstRune(value) == w.Raw {
		return true
	}

	// Check for newlines
	if strings.IndexRune(value, '\n') > -1 {
		return true
	}

	return false
}

// firstRune returns the first rune in the string. Returns 0 for an empty
// string.
func firstRune(s string) rune {
	if s == "" {
		return 0
	}
	return []rune(s)[0]
}

// lastRune returns the last rune in the string. Returns 0 for an empty string.
func lastRune(s string) rune {
	if s == "" {
		return 0
	}
	return []rune(s)[len([]rune(s))-1]
}
