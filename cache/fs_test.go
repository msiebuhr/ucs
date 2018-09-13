package cache

import (
	"bytes"
	"io/ioutil"
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

	// Ends with <key>.info
	suffix := "/cache5.0/00/000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f.info"
	if path[len(path)-len(suffix):] != suffix {
		t.Errorf("Expected suffix '%s' in '%s'", suffix, path)
	}

	// Is not relative
	if path[0] == '.' {
		t.Errorf("Expected path '%s' to be non-relative", path)
	}
}

func TestFSReader(t *testing.T) {
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
	size, reader, err := c.Get(KIND_INFO, key)
	if err != nil {
		t.Fatalf("Unexpected error calling Get(): %#v", err)
	}
	if size != 0 {
		t.Errorf("Expected Get() to return size=0, got %d", size)
	}
	if reader != nil {
		t.Errorf("Got non-nil io.ReadCloser back: %+v", reader)
	}

	// Put non-empty cacheline in
	info := []byte("info")
	cl := Line{Info: &info}

	err = c.Put(key, cl)
	if err != nil {
		t.Fatalf("Unexpected error calling Put(): %s", err)
	}

	// Try again
	size, reader, err = c.Get(KIND_INFO, key)
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
		f.lock.Lock()
		if f.Size != int64(i)+1 {
			t.Errorf("Expected cache size to be %d, got %d", i+1, f.Size)
		}
		f.lock.Unlock()
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
