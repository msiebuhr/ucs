package ucs

import (
	"bytes"
	"testing"
	//"math/rand"
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
}
