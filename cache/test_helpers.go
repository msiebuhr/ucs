package cache

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"
)

// Read from cache and wrap the streaming and size-vs-read-size checks
func readFromCache(cache Cacher, ns string, kind Kind, uuidAndHash []byte) (bool, []byte, error) {
	size, reader, err := cache.Get(ns, kind, uuidAndHash)
	if err != nil {
		return false, []byte{}, err
	}
	// No-hit?
	if size == 0 {
		return false, []byte{}, nil
	}

	if reader == nil {
		return false, []byte{}, fmt.Errorf("Got size, but reader is nil!")
	} else {
		defer reader.Close()
	}

	// Read the data
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return false, []byte{}, err
	}

	if size != int64(len(data)) {
		err = fmt.Errorf("Returned size (%dB) does not match returned data length (%dB)", size, len(data))
		return false, []byte{}, err
	}

	return true, data, nil
}

func testCacheHit(t *testing.T, cache Cacher, ns string, kind Kind, uuidAndHash, expected []byte) {
	hit, data, err := readFromCache(cache, ns, kind, uuidAndHash)
	if err != nil {
		t.Errorf("Unexpected error calling Get(): %#v", err)
		return
	}
	if !hit {
		t.Errorf("Expected hit=%t, got %t", true, hit)
		return
	}
	if !bytes.Equal(data, expected) {
		t.Errorf("Expected '%x', got '%x'", expected, data)
	}
}
