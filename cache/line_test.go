package cache

import (
	"bytes"
	"testing"
)

func TestLineGetHasSet(t *testing.T) {
	kinds := []Kind{KIND_ASSET, KIND_INFO, KIND_RESOURCE}

	for _, kind := range kinds {
		c := Line{}
		t.Run(string(kind), func(t *testing.T) {

			data, ok := c.Get(kind)
			if ok || data != nil {
				t.Errorf("Expected Get('a') to be ([], false), got (%c, %t)", data, ok)
			}

			putData := []byte("X Data goes here!")
			putData[0] = byte(kind)

			err := c.Put(kind, putData)
			if err != nil {
				t.Errorf("Unexpected error Put()'ing: %s", err)
			}

			data, ok = c.Get(kind)
			if !ok || !bytes.Equal(data, putData) {
				t.Errorf("Expected Get('a') to be (['Data goes here!'], true), got (%s, %t)", data, ok)
			}

		})
	}
}
