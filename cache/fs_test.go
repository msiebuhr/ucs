package cache

import (
	"bytes"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
)

func TestFSGenerateFilename(t *testing.T) {
	fs, err := NewFS()
	if err != nil {
		t.Fatalf("Could not create FS: %s", err)
	}
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i % 256)
	}
	path := fs.generateFilename("", KIND_INFO, key)

	// Ends with <key>.info
	suffix := "/unity-cache/__default/00/000102030405060708090a0b0c0d0e0f-101112131415161718191a1b1c1d1e1f.info"
	if path[len(path)-len(suffix):] != suffix {
		t.Errorf("Unexpected suffix\n\t%s\nexpected\n\t%s", suffix, path)
	}

	// Is not relative
	if path[0] == '.' {
		t.Errorf("Expected path '%s' to be non-relative", path)
	}

	// And with namespaces
	path = fs.generateFilename("NameSpace", KIND_INFO, key)

	// Ends with <key>.info
	suffix = "/unity-cache/NameSpace/00/000102030405060708090a0b0c0d0e0f-101112131415161718191a1b1c1d1e1f.info"
	if path[len(path)-len(suffix):] != suffix {
		t.Errorf("Unexpected suffix\n\t%s\nexpected\n\t%s", suffix, path)
	}
}

func TestFSReader(t *testing.T) {
	c, err := NewFS(func(f *FS) { f.Basepath = "./testdata/test-fs-reader/"; f.Quota = 100 })
	if err != nil {
		t.Fatalf("Error creating FS: %s", err)
	}

	defer func() {
		os.RemoveAll(c.Basepath)
	}()

	key := make([]byte, 32)
	rand.Read(key)

	// Negative lookup
	size, reader, err := c.Get("fs", KIND_INFO, key)
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
	tx := c.PutTransaction("fs", key)
	err = tx.Put(int64(len(info)), KIND_INFO, bytes.NewReader(info))
	if err != nil {
		t.Fatalf("Unexpected error calling Put(): %s", err)
	}
	tx.Commit()

	// Try again
	size, reader, err = c.Get("fs", KIND_INFO, key)
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
	f, err := NewFS(func(f *FS) { f.Quota = 100; f.Basepath = "./testdata/fs-quota/" })
	if err != nil {
		t.Fatalf("Error creating FS: %s", err)
	}
	os.RemoveAll(f.Basepath)
	defer func() {
		os.RemoveAll(f.Basepath)
	}()

	// Insert 100 two-byte keys and check the size never gets above 100 bytes.
	for i := 0; i < 100; i++ {
		key := make([]byte, 32)
		rand.Read(key)

		tx := f.PutTransaction("fs", key)
		err := tx.Put(1, KIND_INFO, bytes.NewReader([]byte{byte(i), byte(i)}))
		if err != nil {
			t.Fatalf("Unexpected error calling Put(): %s", err)
		}
		tx.Commit()
	}

	// TODO: Check there is 50 items in the cache.
	f.lock.Lock()
	if f.Size > f.Quota {
		t.Errorf("Expected cache size to be at most %d, got %d", f.Quota, f.Size)
	}
	f.lock.Unlock()

	// Put something large and check it doesn't get bumped immediately
	key := make([]byte, 32)
	rand.Read(key)
	data := make([]byte, 50)
	tx := f.PutTransaction("fs", key)
	tx.Put(int64(len(data)), KIND_INFO, bytes.NewReader(data))
	tx.Commit()

	// Run GC and check size is around 100
	f.collectGarbage()
	if f.Size != 100 {
		t.Errorf("Expected cache size to be 100, has %d", f.Size)
	}

	// Get the last element out again...
	size, _, err := f.Get("fs", KIND_INFO, key)
	if size != int64(len(data)) {
		t.Errorf("Expected to get %d-byte key back, got %db", len(data), size)
	}
}
