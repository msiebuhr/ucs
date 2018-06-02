package ucs

import (
	"fmt"
	"testing"
)

func TestKindStringer(t *testing.T) {
	if fmt.Sprintf("%s", KIND_ASSET) != "a" {
		t.Errorf("Expected KIND_ASSET Stringer to return 'a', got '%s'", KIND_ASSET)
	}
	if fmt.Sprintf("%s", KIND_INFO) != "i" {
		t.Errorf("Expected KIND_INFO Stringer to return 'a', got '%s'", KIND_INFO)
	}
	if fmt.Sprintf("%s", KIND_RESOURCE) != "r" {
		t.Errorf("Expected KIND_RESOURCE Stringer to return 'a', got '%s'", KIND_RESOURCE)
	}
}
