package customflags

import (
	"testing"
)

func TestNamespaces(t *testing.T) {
	// Flag accepts "5000" and "name:5000"
	f := Namespaces{}

	f.Set("5000")
	if val, ok := f[5000]; !ok || val != "5000" {
		t.Errorf("Expected 5000 => 5000, got 5000 => %s", val)
	}

	f.Set("name:6000")
	if val, ok := f[6000]; !ok || val != "name" {
		t.Errorf("Expected 6000 => name, got 6000 => %s", val)
	}
}

func TestFlagNSMulti(t *testing.T) {
	f := Namespaces{}

	f.Set("foo:42,bar:43,baz:44")
	if val, ok := f[42]; !ok || val != "foo" {
		t.Errorf("Expected 42 => foo, got 42 => %s", val)
	}
	if val, ok := f[43]; !ok || val != "bar" {
		t.Errorf("Expected 43 => bar, got 43 => %s", val)
	}
	if val, ok := f[44]; !ok || val != "baz" {
		t.Errorf("Expected 44 => baz, got 44 => %s", val)
	}

	//str := "foo:42 bar:43 baz:44"
	str := "bar:43 baz:44 foo:42"
	out := f.String()
	if out != str {
		t.Errorf("Expected Stringer to return  %s, got %s", str, out)
	}
}

func TestNamespaceMultipleNames(t *testing.T) {
	f := Namespaces{}

	f.Set("foo:3000")
	f.Set("foo:3001")

	expected := "foo:3000 foo:3001"
	if f.String() != expected {
		t.Errorf("Expected %s, got %s", expected, f.String())
	}
}

func TestNamespaceFailsOnPortOverride(t *testing.T) {
	f := Namespaces{}

	f.Set("foo:42")
	err := f.Set("bar:42")

	if err == nil {
		t.Errorf("Expected error when setting same port multiple times")
	}
}
