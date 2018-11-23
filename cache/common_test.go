package cache

import (
	"bytes"
	"math/rand"
	"os"
	"strings"
	"testing"
)

// TODO: Make table-driven
func TestAll(t *testing.T) {
	caches := map[string]Cacher{
		"nop": NewNOP(),
		"mem": NewMemory(1e6),
	}

	c, err := NewFS(func(f *FS) { f.Basepath = "./testdata"; f.Quota = 100 })
	if err != nil {
		t.Fatalf("Error creating FS: %s", err)
	}
	caches["fs"] = c

	for name, cache := range caches {
		t.Run(name, func(t *testing.T) {
			t.Run("namespacing", func(t *testing.T) {
				test_namespacing(t, cache)
			})

			// Write-related tests are skipped for NOP
			if _, ok := cache.(*NOP); ok {
				return
			}

			t.Run("PutTransaction", func(t *testing.T) {
				test_commit_transaction(t, cache)
			})
		})
	}

	defer func() {
		os.RemoveAll(c.Basepath)
	}()
}

func test_namespacing(t *testing.T, c Cacher) {
	key := make([]byte, 32)
	rand.Read(key)

	// Negative lookup
	size, reader, err := c.Get("a", KIND_INFO, key)
	if err != nil {
		t.Fatalf("Unexpected error calling Get(): %#v", err)
	}
	if size != 0 {
		t.Errorf("Expected Get() to return size=0, got %d", size)
	}
	if reader != nil {
		t.Errorf("Got non-nil io.ReadCloser back: %+v", reader)
	}

	// Put things in 'a' namespaace
	info := []byte("info")
	tx := c.PutTransaction("a", key)
	tx.Put(int64(len(info)), KIND_INFO, bytes.NewReader(info))
	tx.Commit()

	// Negative lookup for key in 'b'
	size, reader, err = c.Get("b", KIND_INFO, key)
	if err != nil {
		t.Fatalf("Unexpected error calling Get(): %#v", err)
	}
	if size != 0 {
		t.Errorf("Expected Get() to return size=0, got %d", size)
	}
	if reader != nil {
		t.Errorf("Got non-nil io.ReadCloser back: %+v", reader)
	}
}

func test_commit_transaction(t *testing.T, c Cacher) {
	key := make([]byte, 32)
	rand.Read(key)

	// Put wia a transaction
	tx := c.PutTransaction("tx", key)
	err := tx.Put(6, KIND_INFO, strings.NewReader("foobar"))
	if err != nil {
		t.Fatalf("Unexpected error calling Put(): %#v", err)
	}

	// Commit
	// TODO: Try doing a read before this!
	tx.Commit()

	// Positive lookup for `key`
	testCacheHit(t, c, "tx", KIND_INFO, key, []byte("foobar"))
}
