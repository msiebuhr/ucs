package cache

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

type FS struct {
	lock     sync.RWMutex
	Basepath string
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

func (fs FS) generatePath(kind Kind, uuidAndHash []byte) string {
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
