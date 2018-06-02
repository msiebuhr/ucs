package ucs

import (
	"log"
	"sync"
)

type CacheMemory struct {
	lock sync.RWMutex
	data map[string]CacheLine
}

func NewCacheMemory() *CacheMemory {
	return &CacheMemory{data: make(map[string]CacheLine)}
}

func (c *CacheMemory) Has(kind byte, uuidAndHash []byte) (bool, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	log.Printf("CacheMemory.Has %c %s", kind, PrettyUuidAndHash(uuidAndHash))
	if entry, ok := c.data[string(uuidAndHash)]; ok {
		return entry.Has(kind), nil
	}
	return false, nil
}

func (c *CacheMemory) Put(uuidAndHash []byte, data CacheLine) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	log.Printf("CacheMemory.Put %s", PrettyUuidAndHash(uuidAndHash))
	c.data[string(uuidAndHash)] = data

	return nil
}

func (c *CacheMemory) Get(kind byte, uuidAndHash []byte) ([]byte, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	log.Printf("CacheMemory.Get %c %s", kind, PrettyUuidAndHash(uuidAndHash))

	if data, ok := c.data[string(uuidAndHash)]; ok {
		bytes, _ := data.Get(kind)
		return bytes, nil
	}

	return []byte{}, nil
}
