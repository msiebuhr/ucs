package ucs

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
