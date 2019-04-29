package ucs

import (
	"fmt"
	"io"
	"testing"
)

func TestRandomStreams(t *testing.T) {

	var tests = []struct {
		reader      io.Reader
		writer      *RandWriter
		bytesTested int64
		err         error
	}{
		{
			NewRandReader(0xdeadbeef, 1e5),
			NewRandWriter(0xdeadbeef, 1e5),
			1e5,
			nil,
		},
		// Send more data than expected
		{
			NewRandReader(42, 1000),
			NewRandWriter(42, 999),
			1000,
			//fmt.Errorf("Expected 999 bytes, got 1000"),
			fmt.Errorf("Different data detected at byte 999"),
		},
		// Less data than expected
		{
			NewRandReader(42, 10),
			NewRandWriter(42, 100),
			10,
			fmt.Errorf("Expected 100 bytes, got 10"),
		},
		// Different stream
		{
			NewRandReader(1, 1e6),
			NewRandWriter(2, 1e6),
			1e6,
			fmt.Errorf("Different data detected at byte 0"),
		},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			n, err := io.Copy(tt.writer, tt.reader)
			if err != nil {
				t.Errorf("Expected no error, got '%s'", err)
			}
			if n != tt.bytesTested {
				t.Errorf("Expected %d bytes copied, did %d", tt.bytesTested, n)
			}

			if tt.err == nil && tt.writer.Done() != nil {
				t.Errorf("Unexpected error from Done(): %s", tt.writer.Done())
			} else if tt.err != nil && tt.writer.Done() == nil {
				t.Errorf("Expected error '%s' got none", tt.err)
			} else if tt.err != nil && tt.writer.Done() != nil && tt.err.Error() != tt.writer.Done().Error() {
				t.Errorf("Expected error '%s', got '%s'", tt.err, tt.writer.Done().Error())
			}
		})
	}
}
