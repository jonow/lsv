package lsv

import (
	"reflect"
	"testing"
)

// Tests that SplitParams returns the expected error or value for each test.
func TestSplitParams(t *testing.T) {
	newParameters := func(tt readTest) Parameters {
		p := DefaultParameters()

		if tt.Comment != 0 {
			p.Comment = tt.Comment
		}
		if tt.Raw != 0 {
			p.Raw = tt.Raw
		}
		if tt.Escape != 0 {
			p.Escape = tt.Escape
		}
		if tt.NoTrim {
			p.TrimLeadingSpace = false
		}
		return p
	}

	for _, tt := range readTests {
		t.Run(tt.Name, func(t *testing.T) {
			p := newParameters(tt)
			out, err := SplitParams(tt.Input, p)
			if tt.Error != nil {
				if !reflect.DeepEqual(err, tt.Error) {
					t.Fatalf("SplitParams() error mismatch:"+
						"\nexpected: %v (%#v)\nreceived: %v (%#v)",
						tt.Error, tt.Error, err, err)
				}
				if out != nil {
					t.Fatalf("SplitParams() unexpected output:"+
						"\nexpected: nil\nreceived: %q", out)
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected SplitParams() error: %+v", err)
				}
				if !reflect.DeepEqual(out, tt.Output) {
					t.Fatalf("SplitParams() unexpected output:"+
						"\nexpected: %q\nreceived: %q", tt.Output, out)
				}
			}
		})
	}
}
