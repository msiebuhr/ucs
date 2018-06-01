package ucs

import (
	"bytes"
	"math/rand"
	"testing"
)

func TestCacheLineGetHasSet(t *testing.T) {
	kinds := []byte{'a', 'i', 'r'}

	for _, kind := range kinds {
		c := CacheLine{}
		t.Run(string(kind), func(t *testing.T) {

			ok := c.Has(kind)
			if ok {
				t.Errorf("Excpected Has('a') to be false, got %t", ok)
			}

			data, ok := c.Get(kind)
			if ok || data != nil {
				t.Errorf("Expected Get('a') to be ([], false), got (%c, %t)", data, ok)
			}

			putData := []byte("X Data goes here!")
			putData[0] = kind

			err := c.Put(kind, putData)
			if err != nil {
				t.Errorf("Unexpected error Put()'ing: %s", err)
			}

			ok = c.Has(kind)
			if !ok {
				t.Errorf("Excpected Has('a') to be true, got %t", ok)
			}

			data, ok = c.Get(kind)
			if !ok || !bytes.Equal(data, putData) {
				t.Errorf("Expected Get('a') to be (['Data goes here!'], true), got (%s, %t)", data, ok)
			}

		})
	}
}

func TestCacheMemorySimple(t *testing.T) {
	c := NewCacheMemory()
	key := make([]byte, 32)
	rand.Read(key)

	// Negative lookup
	ok, err := c.Has(TYPE_INFO, key)
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
	ok, err = c.Has(TYPE_INFO, key)
	if err != nil {
		t.Fatalf("Unexpected error calling Has(): %s", err)
	}
	if !ok {
		t.Errorf("Expected Has() to return true, got %t", ok)
	}
}
