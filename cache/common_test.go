package cache

import (
	"bytes"
	"math/rand"
	"os"
	"testing"
)

// TODO: Make table-driven
func TestCommon(t *testing.T) {
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
			t.Run("Namespaces", func(t *testing.T) {
				test_namespacing(t, cache)
			})

			t.Run("AdminDeleteNop", func(t *testing.T) {
				count, err := cache.Remove("nop", []byte("nopnopnop"))
				if err != nil || (count != 0) {
					t.Errorf("Expected Delete('nopnopnop') to NOP, got %d, %s", count, err)
				}
			})

			// Skip these for the NOP cache
			if _, ok := cache.(*NOP); ok {
				return
			}

			t.Run("Search", func(t *testing.T) {
				test_search_and_remove(t, cache)
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
	cl := Line{Info: &info}

	err = c.Put("a", key, cl)
	if err != nil {
		t.Fatalf("Unexpected error calling Put(): %s", err)
	}

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

func test_search_and_remove(t *testing.T, c Cacher) {
	key := make([]byte, 32)
	rand.Read(key)

	// Add something to search for!
	info := []byte("info")
	cl := Line{Info: &info}

	err := c.Put("search", key, cl)
	if err != nil {
		t.Fatalf("Unexpected error calling Put(): %s", err)
	}

	// Negative lookup
	ch, err := c.Search("search", []byte("info-missing"))
	if err != nil {
		t.Fatalf("Unexpected error calling Search(): %#v", err)
	}

	for result := range ch {
		t.Errorf("Unexpected result from NO-OP search: %v", result)
	}

	// Positive lookup
	ch, err = c.Search("search", key[:16])
	if err != nil {
		t.Fatalf("Unexpected error calling Search(): %#v", err)
	}

	results := 0
	for result := range ch {
		results += 1
		if !bytes.Equal(result.UuidAndHash, key) {
			t.Errorf("Expected UuidAndHash to be %x, got %x", key, result.UuidAndHash)
		}
		if result.InfoSize != 4 {
			t.Errorf("Expected InfoSize to be %d, got %d", 4, result.InfoSize)
		}
		if result.AssetSize != 0 {
			t.Errorf("Expected InfoSize to be %d, got %d", 0, result.AssetSize)
		}
		if result.ResourceSize != 0 {
			t.Errorf("Expected InfoSize to be %d, got %d", 0, result.ResourceSize)
		}
	}

	if results != 1 {
		t.Errorf("Expected Search() to yield one result, got %d", results)
	}

	// Now remove the key
	deleted, err := c.Remove("search", key)
	if deleted != 1 || err != nil {
		t.Errorf("Expected Remove() to return (1, nil), got (%d, %v)", deleted, err)
	}

	// Delete again and get nothing
	deleted, err = c.Remove("search", key)
	if deleted != 0 || err != nil {
		t.Errorf("Expected Remove() to return (0, nil), got (%d, %v)", deleted, err)
	}

}
