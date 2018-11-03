package customflags

import (
	"testing"
)

func TestNamespaces(t *testing.T) {
	// Flag accepts "5000" and "name:5000"
	f := Namespaces{}

	f.Set("5000")
	if val, ok := f["5000"]; !ok || val != 5000 {
		t.Errorf("Expected 5000 => 5000, got 5000 => %d", val)
	}

	f.Set("name:6000")
	if val, ok := f["name"]; !ok || val != 6000 {
		t.Errorf("Expected name => 6000, got name => %d", val)
	}
}

func TestFlagNSMulti(t *testing.T) {
	f := Namespaces{}

	f.Set("foo:42,bar:43,baz:44")
	if val, ok := f["foo"]; !ok || val != 42 {
		t.Errorf("Expected foo => 42, got foo => %d", val)
	}
	if val, ok := f["bar"]; !ok || val != 43 {
		t.Errorf("Expected bar => 43, got bar => %d", val)
	}
	if val, ok := f["baz"]; !ok || val != 44 {
		t.Errorf("Expected baz => 44, got baz => %d", val)
	}
}

func TestSize(t *testing.T) {
	s := NewSize(1e6)

	s.Set("1KB")
	if s.Int64() != 1000 /*1024*/ {
		t.Errorf("Expected '1K' => 1000 bytes, got %d", s.Int64());
	}
}
