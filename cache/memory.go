package cache

import (
	"sync"
)

type Memory struct {
	lock sync.RWMutex
	data map[string]Line
}

func NewMemory() *Memory {
	return &Memory{data: make(map[string]Line)}
}

func (c *Memory) Put(uuidAndHash []byte, data Line) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.data[string(uuidAndHash)] = data

	return nil
}

func (c *Memory) Get(kind Kind, uuidAndHash []byte) ([]byte, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if data, ok := c.data[string(uuidAndHash)]; ok {
		bytes, _ := data.Get(kind)
		return bytes, nil
	}

	return []byte{}, nil
}
