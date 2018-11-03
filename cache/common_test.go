package cache

import (
	"math/rand"
	"os"
	"testing"
)

// TODO: Make table-driven
func TestNamespacing(t *testing.T) {
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
			test_namespacing(t, cache)
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
