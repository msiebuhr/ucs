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
	c, err := NewFS(func(f *FS) { f.Basepath = "./testdata"; f.Quota = 100 })
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

func TestFSQuota(t *testing.T) {
	f, err := NewFS(func(f *FS) { f.Quota = 100; f.Basepath = "./testdata" })
	if err != nil {
		t.Fatalf("Error creating FS: %s", err)
	}

	keys := make([][]byte, 100)

	// Insert 100 keys
	for i := 0; i < len(keys); i++ {
		keys[i] = make([]byte, 32)
		rand.Read(keys[i])

		cl := Line{Info: &[]byte{byte(i)}}
		err := f.Put(keys[i], cl)
		if err != nil {
			t.Fatalf("Unexpected error calling Put(): %s", err)
		}

		// Run the garbage collector explicitly
		f.collectGarbage()
		if f.Size != int64(i)+1 {
			t.Errorf("Expected cache size to be %d, got %d", i+1, f.Size)
		}
	}

	// Put something large and check it is bumps everything else off
	data := make([]byte, 100)
	cl := Line{Info: &data}
	f.Put(make([]byte, 32), cl)

	// Run GC and check size is around 100
	f.collectGarbage()
	//if len(n.data) != 1 {
	//	t.Errorf("Expected cache length to be 1, has %d", len(c.data))
	//}
}
