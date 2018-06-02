package ucs

import (
	"math/rand"
	"testing"
)

func TestCacheMemorySimple(t *testing.T) {
	c := NewCacheMemory()
	key := make([]byte, 32)
	rand.Read(key)

	// Negative lookup
	ok, err := c.Has(KIND_INFO, key)
	if err != nil {
		t.Fatalf("Unexpected error calling Has(): %s", err)
	}
	if ok {
		t.Errorf("Expected Has() to return false, got %t", ok)
	}

	// Put non-empty cacheline in
	info := []byte("info")
	data := CacheLine{Info: &info}

	err = c.Put(key, data)
	if err != nil {
		t.Fatalf("Unexpected error calling Put(): %s", err)
	}

	// Try again
	ok, err = c.Has(KIND_INFO, key)
	if err != nil {
		t.Fatalf("Unexpected error calling Has(): %s", err)
	}
	if !ok {
		t.Errorf("Expected Has() to return true, got %t", ok)
	}
}
