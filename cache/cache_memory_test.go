package cache

import (
	"bytes"
	"math/rand"
	"testing"
)

func TestCacheMemorySimple(t *testing.T) {
	c := NewCacheMemory()
	key := make([]byte, 32)
	rand.Read(key)

	// Negative lookup
	data, err := c.Get(KIND_INFO, key)
	if err != nil {
		t.Fatalf("Unexpected error calling Get(): %s", err)
	}
	if len(data) != 0 {
		t.Errorf("Expected Get() to return '' got '%s'", data)
	}

	// Put non-empty cacheline in
	info := []byte("info")
	cl := CacheLine{Info: &info}

	err = c.Put(key, cl)
	if err != nil {
		t.Fatalf("Unexpected error calling Put(): %s", err)
	}

	// Try again
	data, err = c.Get(KIND_INFO, key)
	if err != nil {
		t.Fatalf("Unexpected error calling Has(): %s", err)
	}
	if !bytes.Equal(data, info) {
		t.Errorf("Expected Get() to return %s, got %s", info, data)
	}
}
