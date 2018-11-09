package cache

import (
	"io"
)

// Denotes which kind of data goes in the cache
type Kind byte

const (
	KIND_ASSET    Kind = 'a'
	KIND_INFO     Kind = 'i'
	KIND_RESOURCE Kind = 'r'
)

func (k Kind) String() string {
	return string(k)
}

type LineInfo struct {
	UuidAndHash  []byte
	InfoSize     int
	AssetSize    int
	ResourceSize int
}

// Cacher is the interface to be implemented by caches
type Cacher interface {
	// Put a cache-line with the given namespace and uuid/hash
	Put(string, []byte, Line) error

	// Get an asset based on its namespace, kind and uuid/hash
	// combination. Returns the asset size, reader and error.
	Get(string, Kind, []byte) (int64, io.ReadCloser, error)

	// Searches the cache for items with the given prefix and sends back found instances
	Search(string, []byte) (<-chan LineInfo, error)

	// Remove a specific UUID/Hash combination from the cache. Returns the
	// number of delete entries + any error.
	Remove(string, []byte) (int, error)
}
