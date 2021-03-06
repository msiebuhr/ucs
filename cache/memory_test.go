package cache

import (
	"bytes"
	"io/ioutil"
	"math/rand"
	"testing"
)

func TestMemoryReader(t *testing.T) {
	c := NewMemory(1e6)
	key := make([]byte, 32)
	rand.Read(key)

	// Negative lookup
	size, reader, err := c.Get("mem", KIND_INFO, key)
	if err != nil {
		t.Fatalf("Unexpected error calling Get(): %s", err)
	}
	if size > 0 {
		t.Errorf("Expected Get() to return 0, got %d", size)
	}
	if reader != nil {
		t.Errorf("Got non-nil io.ReadCloser back: %+v", reader)
	}

	// Put non-empty cacheline in
	info := []byte("info")
	tx := c.PutTransaction("mem", key)
	tx.Put(int64(len(info)), KIND_INFO, bytes.NewReader(info))
	tx.Commit()

	// Try again
	size, reader, err = c.Get("mem", KIND_INFO, key)
	if err != nil {
		t.Fatalf("Unexpected error calling Get(): %s", err)
	}
	if size != int64(len(info)) {
		t.Errorf("Expected Get() to return size=%d, got %d", len(info), size)
	}
	if reader == nil {
		t.Error("Got nil io.ReadCloser back...", reader)
	}
	defer reader.Close()

	data, err := ioutil.ReadAll(reader)
	if err != nil {
		t.Fatalf("Unexpected error reading returned data: %s", err)
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

		info := []byte{byte(i)}
		tx := c.PutTransaction("mem", keys[i])
		err := tx.Put(int64(len(info)), KIND_INFO, bytes.NewReader(info))
		if err != nil {
			t.Fatalf("Unexpected error calling Put(): %s", err)
		}
		tx.Commit()

		if c.size != int64(i)+1 {
			t.Errorf("Expected cache size to be %d, got %d", i+1, c.size)
		}
	}

	// Put something large and check it is bumps everything else off
	data := make([]byte, 100)
	tx := c.PutTransaction("mem", make([]byte, 32))
	tx.Put(100, KIND_INFO, bytes.NewReader(data))
	tx.Commit()

	if len(c.data) != 1 {
		t.Errorf("Expected cache length to be 1, has %d", len(c.data))
	}
}
