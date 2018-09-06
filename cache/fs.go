package cache

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type FS struct {
	lock     sync.RWMutex
	Basepath string
	Size     int64
	Quota    int64
}

func NewFS(options ...func(*FS)) (*FS, error) {
	fs := &FS{Basepath: "./cache"}
	for _, f := range options {
		f(fs)
	}

	// Make sure FS is an absolute path
	path, err := filepath.Abs(fs.Basepath)
	if err != nil {
		return fs, err
	}
	fs.Basepath = path

	return fs, nil
}

// Remove old files (as measured by the most recent ATIME of any entry sharing
// the same UUID/hash)
// Also re-calculates the total size of cache directory, now we're at scanning
// everything anyway...
func (fs *FS) collectGarbage() {
	fs.lock.Lock()
	defer fs.lock.Unlock()

	var old = make([]struct {
		uuidAndHash string
		size        int64
		time        time.Time
	}, 256)

	fs.Size = 0

	// There be 256 folders - let's find the oldest one + it's size
	// TODO: Split into ~256 go-routines for speed
	for i := 0; i < 256; i += 1 {
		dirname := filepath.Join(fs.Basepath, fmt.Sprintf("%02x", i))
		dir, err := os.Open(dirname)
		if err != nil {
			continue
		}
		entries, err := dir.Readdir(0)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			// Count up sizes of everything
			fs.Size += entry.Size()

			// Check if it's the oldest thing we've found in this directory
			t := fileinfo_atime(entry)
			if len(old[i].uuidAndHash) == 0 || t.Before(old[i].time) {
				old[i].uuidAndHash = entry.Name() // TODO: Chop off extension
				old[i].size = entry.Size()
				old[i].time = t
			}
		}
	}

	// Ideally, we should delete the very oldest stuff first (and both info and
	// asset/resource), and then re-scan that directory.
	// But I'm lazy right now - let's just delete the oldst thing we found in
	// all folders and see how far that get's us.
	for i := 0; i < 256 && fs.Size > fs.Quota; i += 1 {
		if old[i].uuidAndHash == "" {
			continue
		}

		path := filepath.Join(fs.Basepath, fmt.Sprintf("%02x", i), old[i].uuidAndHash)

		err := os.Remove(path)
		if err == nil {
			fs.Size -= old[i].size
		}

		// Bail if we get below the quota
		if fs.Size <= fs.Quota {
			return
		}
	}

	// If we're still over quota, do another round of GC'ing
	if fs.Size > fs.Quota {
		go fs.collectGarbage()
	}
}

func (fs *FS) generatePath(kind Kind, uuidAndHash []byte) string {
	return filepath.Join(fs.Basepath, fmt.Sprintf("%02x", uuidAndHash[:1]), fmt.Sprintf("%032x.%c", uuidAndHash, kind))
}

func (fs *FS) putKind(kind Kind, uuidAndHash, data []byte) error {
	path := fs.generatePath(kind, uuidAndHash)

	//fs.lock.Lock()
	//defer fs.lock.Unlock()

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	f.Write(data)

	return nil
}

func (fs *FS) Put(uuidAndHash []byte, data Line) error {
	fs.lock.Lock()
	defer fs.lock.Unlock()

	// Kick of GC if we're above the size
	fs.Size += data.Size()
	if fs.Size > fs.Quota {
		go fs.collectGarbage()
	}

	// Make sure leading directory exists!
	leadingPath := filepath.Join(fs.Basepath, fmt.Sprintf("%02x", uuidAndHash[:1]))
	os.MkdirAll(leadingPath, os.ModePerm)

	// Loop over types in the Put
	if data.Info != nil {
		err := fs.putKind(KIND_INFO, uuidAndHash, *data.Info)
		if err != nil {
			return err
		}
	}
	if data.Resource != nil {
		err := fs.putKind(KIND_RESOURCE, uuidAndHash, *data.Resource)
		if err != nil {
			return err
		}
	}
	if data.Asset != nil {
		err := fs.putKind(KIND_ASSET, uuidAndHash, *data.Asset)
		if err != nil {
			return err
		}
	}

	return nil
}

func (fs *FS) Get(kind Kind, uuidAndHash []byte) ([]byte, error) {
	path := fs.generatePath(kind, uuidAndHash)

	fs.lock.RLock()
	defer fs.lock.RUnlock()

	f, err := os.Open(path)
	if err != nil {
		return []byte{}, nil
	}
	defer f.Close()

	return ioutil.ReadAll(f)
}
