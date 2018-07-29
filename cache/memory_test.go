package cache

import (
	"bytes"
	"math/rand"
	"testing"
)

func TestMemorySimple(t *testing.T) {
	c := NewMemory(1e6)
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
	cl := Line{Info: &info}

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

func TestMemoryquota(t *testing.T) {
	c := NewMemory(100)
	keys := make([][]byte, 100)

	// Insert 100 keys
	for i := 0; i < len(keys); i++ {
		keys[i] = make([]byte, 32)
		rand.Read(keys[i])

		cl := Line{Info: &[]byte{byte(i)}}
		err := c.Put(keys[i], cl)
		if err != nil {
			t.Fatalf("Unexpected error calling Put(): %s", err)
		}

		if c.size != i+1 {
			t.Errorf("Expected cache size to be %d, got %d", i+1, c.size)
		}
	}

	// Put something large and check it is bumps everything else off
	data := make([]byte, 100)
	cl := Line{Info: &data}
	c.Put(make([]byte, 32), cl)

	if len(c.data) != 1 {
		t.Errorf("Expected cache length to be 1, has %d", len(c.data))
	}
}
