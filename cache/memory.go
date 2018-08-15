package cache

import (
	"sync"
)

type memoryEntry struct {
	data       map[Kind][]byte
	generation int
	size       int64
}

func memoryEntryFromLine(generation int, line Line) memoryEntry {
	m := memoryEntry{
		generation: generation,
		data:       make(map[Kind][]byte),
		size:       0,
	}

	if data, ok := line.Get(KIND_ASSET); ok {
		m.data[KIND_ASSET] = data
		m.size += int64(len(data))
	}
	if data, ok := line.Get(KIND_INFO); ok {
		m.data[KIND_INFO] = data
		m.size += int64(len(data))
	}
	if data, ok := line.Get(KIND_RESOURCE); ok {
		m.data[KIND_RESOURCE] = data
		m.size += int64(len(data))
	}

	return m
}

func (m *memoryEntry) Get(kind Kind) ([]byte, bool) {
	data, ok := m.data[kind]
	return data, ok
}

type Memory struct {
	lock sync.RWMutex
	data map[string]memoryEntry

	// Track current size, quota
	size  int64
	quota int64

	// Monotonically increasing counter to track age of objects
	generation int
}

func NewMemory(quota int64) *Memory {
	return &Memory{quota: quota, data: make(map[string]memoryEntry)}
}

func (m *Memory) collectGarbage(spaceToMake int64) {
	//m.lock.Lock()
	//defer m.lock.Unlock()

	// TODO: Make sure we don't get stuck in infinite loops

	// Walk all keys and delete the oldest data until we have room...
	for spaceToMake+m.size > m.quota {
		oldestGeneration := m.generation + 1
		oldestKey := ""

		for key, data := range m.data {
			if data.generation < oldestGeneration {
				oldestGeneration = data.generation
				oldestKey = key
			}
		}

		// Decrement size and remove key
		m.size -= m.data[oldestKey].size
		delete(m.data, oldestKey)
	}
}

func (c *Memory) Put(uuidAndHash []byte, data Line) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.generation++

	line := memoryEntryFromLine(c.generation, data)

	c.collectGarbage(line.size)

	c.data[string(uuidAndHash)] = line
	c.size += line.size

	return nil
}

func (c *Memory) Get(kind Kind, uuidAndHash []byte) ([]byte, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	line, ok := c.data[string(uuidAndHash)]

	if !ok {
		return []byte{}, nil
	}

	if bytes, ok := line.data[kind]; ok {
		// Update generation
		c.generation++
		line.generation = c.generation

		return bytes, nil
	}

	return []byte{}, nil
}
