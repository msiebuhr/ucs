package cache

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

// Cacher is the interface to be implemented by caches
type Cacher interface {
	Put([]byte, Line) error
	Get(Kind, []byte) ([]byte, error)
}
