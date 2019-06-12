package customflags

import (
	"testing"
)

func TestSize(t *testing.T) {
	s := NewSize(1e6)

	s.Set("1KB")
	if s.Int64() != 1024 {
		t.Errorf("Expected '1K' => 1024 bytes, got %d", s.Int64())
	}

	if s.String() != "1KiB" {
		t.Errorf("Expected '1KB'.String() => 1KiB, got %s", s.String())
	}
}
