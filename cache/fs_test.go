package cache

import (
	"bytes"
	"math/rand"
	"os"
	"testing"
)

func TestFSGeneratePath(t *testing.T) {
	fs, err := NewFS()
	if err != nil {
		t.Fatalf("Could not create FS: %s", err)
	}
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i % 256)
	}
	path := fs.generatePath(KIND_INFO, key)

	// Ends with <key>.i
	suffix := "/cache/00/000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f.i"
	if path[len(path)-len(suffix):] != suffix {
		t.Errorf("Expected suffix '%s' in '%s'", suffix, path)
	}

	// Is not relative
	if path[0] == '.' {
		t.Errorf("Expected path '%s' to be non-relative", path)
	}
}

func TestFSSimple(t *testing.T) {
	c, err := NewFS(func(f *FS) { f.Basepath = "./testdata" })
	if err != nil {
		t.Fatalf("Error creating FS: %s", err)
	}

	defer func() {
		os.RemoveAll(c.Basepath)
	}()

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
