package ucs

import (
	"context"
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

func (c CacheLine) Get(kind byte) ([]byte, bool) {
	switch kind {
	case TYPE_ASSET:
		return *c.Asset, c.Asset != nil
	case TYPE_INFO:
		return *c.Info, c.Info != nil
	case TYPE_RESOURCE:
		return *c.Resource, c.Resource != nil
	}
	return nil, false
}

func (c CacheLine) Has(kind byte) bool {
	switch kind {
	case TYPE_ASSET:
		return c.Asset != nil
	case TYPE_INFO:
		return c.Info != nil
	case TYPE_RESOURCE:
		return c.Resource != nil
	default:
		return false
	}
}

func (c *CacheLine) Put(kind byte, data []byte) error {
	log.Printf("CacheLine.Put %c %x", kind, data)
	switch kind {
	case TYPE_ASSET:
		c.Asset = &data
	case TYPE_INFO:
		c.Info = &data
	case TYPE_RESOURCE:
		c.Resource = &data
	default:
		return errors.New("Trying to put unknown resource")
	}
	return nil
}

// Put data from a reader into the cacheline. The kind and number of bytes
// to be read as well
func (c *CacheLine) PutReader(kind byte, size uint64, r io.Reader) error {
	log.Printf("CacheLine.PutReader %c %db", kind, size)
	tmp := make([]byte, size)
	_, err := io.ReadFull(r, tmp)
	if err != nil {
		return err
	}
	return c.Put(kind, tmp)
}

type CacheMemory struct {
	data map[string]CacheLine
}

func NewCacheMemory(ctx context.Context) *CacheMemory {
	return &CacheMemory{data: make(map[string]CacheLine)}
}

func (c *CacheMemory) Has(kind byte, uuidAndHash []byte) (bool, error) {
	log.Printf("CacheMemory.Has %c %s", kind, PrettyUuidAndHash(uuidAndHash))
	if entry, ok := c.data[string(uuidAndHash)]; ok {
		return entry.Has(kind), nil
	}
	return false, nil
}

func (c *CacheMemory) Put(uuidAndHash []byte, data CacheLine) error {
	log.Printf("CacheMemory.Put %s", PrettyUuidAndHash(uuidAndHash))
	c.data[string(uuidAndHash)] = data

	return nil
}

func (c *CacheMemory) Get(kind byte, uuidAndHash []byte) ([]byte, error) {
	log.Printf("CacheMemory.Get %c %s", kind, PrettyUuidAndHash(uuidAndHash))

	if data, ok := c.data[string(uuidAndHash)]; ok {
		bytes, _ := data.Get(kind)
		return bytes, nil
	}

	return []byte{}, nil
}
