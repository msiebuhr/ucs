package cache

import (
	"errors"
	"io"
	"log"
)

type CacheLine struct {
	// TODO: Do we really need a self-reference?
	uuidAndHash []byte

	// Various kinds of data...
	Asset    *[]byte
	Info     *[]byte
	Resource *[]byte
}

func (c CacheLine) Get(kind Kind) ([]byte, bool) {
	var ptr *[]byte
	switch kind {
	case KIND_ASSET:
		ptr = c.Asset
	case KIND_INFO:
		ptr = c.Info
	case KIND_RESOURCE:
		ptr = c.Resource
	default:
		return []byte{}, false
	}

	if ptr == nil {
		return nil, false
	}

	return *ptr, true
}

func (c *CacheLine) Put(kind Kind, data []byte) error {
	log.Printf("CacheLine.Put %c %dB", kind, len(data))
	switch kind {
	case KIND_ASSET:
		c.Asset = &data
	case KIND_INFO:
		c.Info = &data
	case KIND_RESOURCE:
		c.Resource = &data
	default:
		return errors.New("Trying to put unknown resource")
	}
	return nil
}

// Put data from a reader into the cacheline. The kind and number of bytes
// to be read as well
func (c *CacheLine) PutReader(kind Kind, size uint64, r io.Reader) error {
	log.Printf("CacheLine.PutReader %c %db", kind, size)
	tmp := make([]byte, size)
	_, err := io.ReadFull(r, tmp)
	if err != nil {
		return err
	}
	return c.Put(kind, tmp)
}
