package lsv

import (
	"bytes"
	"encoding/csv"
	"io"
	"strings"
	"testing"
)

func Benchmark_Split(b *testing.B) {
	line := "this is a value   # This is a comment\n"
	var str strings.Builder
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		str.WriteString(line)
		buff := bytes.NewBufferString("")
		buff.WriteString(line)
		b.StartTimer()
		data, _ := io.ReadAll(buff)
		_, err := Split(string(data))
		if err != nil {
			b.Errorf("splitter error: %+v", err)
		}
	}
}

func Benchmark_ReadAll(b *testing.B) {
	line := "this is a value   # This is a comment\n"
	var str strings.Builder
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		str.WriteString(line)
		strReader := strings.NewReader(str.String())
		r := NewReader(strReader)
		b.StartTimer()
		_, err := r.ReadAll()
		if err != nil {
			b.Errorf("splitter error: %+v", err)
		}
	}
}

func Benchmark_CSV(b *testing.B) {
	line := "this is a value   # This is a comment\n"
	var str strings.Builder
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		str.WriteString(line)
		strReader := strings.NewReader(str.String())
		r := csv.NewReader(strReader)
		b.StartTimer()
		_, err := r.ReadAll()
		if err != nil {
			b.Errorf("splitter error: %+v", err)
		}
	}
}
