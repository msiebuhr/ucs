package cache

import (
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"

	"github.com/docker/go-units"
)

func benchmarkBackendSequentialReadBuf(b *testing.B, c Cacher, size int64) {
	key := make([]byte, 32)
	rand.Read(key)

	// Put non-empty cacheline in
	info := make([]byte, size)
	rand.Read(info)
	cl := Line{Info: &info}

	err := c.Put("bench", key, cl)
	if err != nil {
		b.Fatalf("Unexpected error calling Put(): %s", err)
	}

	b.SetBytes(size)
	b.ResetTimer()

	// Try again
	for i := 0; i < b.N; i += 1 {
		size, reader, err := c.Get("bench", KIND_INFO, key)
		if err != nil {
			b.Fatalf("Unexpected error calling Has(): %s", err)
		}
		if size == 0 {
			b.Errorf("Expected Get() to return %d, got %d", 0, size)
		}
		if reader != nil {
			io.Copy(ioutil.Discard, reader)
			reader.Close()
		}
	}
}

func BenchmarkFSPositive(b *testing.B) {
	c, err := NewFS(func(f *FS) {
		f.Basepath = "./testdata"
		f.Quota = 1024 * 1024 * 1024
	})
	if err != nil {
		b.Fatalf("Error creating FS: %s", err)
	}

	defer func() {
		os.RemoveAll(c.Basepath)
	}()

	for _, size := range []int64{1024, 1024 * 128, 1024 * 1024, 1024 * 1024 * 128} {
		b.Run(fmt.Sprintf("streaming,size=%s", units.BytesSize(float64(size))), func(b *testing.B) {
			benchmarkBackendSequentialReadBuf(b, c, size)
		})
	}
}

func BenchmarkFSPositiveStream(b *testing.B) {
	c, err := NewFS(func(f *FS) {
		f.Basepath = "./testdata"
		f.Quota = 1024 * 10
	})
	if err != nil {
		b.Fatalf("Error creating FS: %s", err)
	}

	defer func() {
		os.RemoveAll(c.Basepath)
	}()

	key := make([]byte, 32)
	rand.Read(key)

	// Put non-empty cacheline in
	info := make([]byte, 1024)
	rand.Read(info)
	cl := Line{Info: &info}

	err = c.Put("bench", key, cl)
	if err != nil {
		b.Fatalf("Unexpected error calling Put(): %s", err)
	}

	b.SetBytes(int64(len(info)))
	b.ResetTimer()

	// Try again
	for i := 0; i < b.N; i += 1 {
		size, reader, err := c.Get("bench", KIND_INFO, key)
		if err != nil {
			b.Fatalf("Unexpected error calling Has(): %s", err)
		}
		if size == 0 {
			b.Errorf("Expected Get() to return %d, got %d", 0, size)
		}
		io.Copy(ioutil.Discard, reader)
		reader.Close()
	}
}
