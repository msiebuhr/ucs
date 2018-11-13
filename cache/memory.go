package cache

import (
	"bytes"
	"io"
	"io/ioutil"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	memory_gc_duration = prometheus.NewSummary(prometheus.SummaryOpts{
		Name: "ucs_memorycache_gc_duration_seconds",
		Help: "Time spent deleting data",
	})
	memory_gc_bytes = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "ucs_memorycache_gc_removed_bytes",
		Help: "Bytes deleted by GC",
	})
)

func init() {
	prometheus.MustRegister(memory_gc_duration)
	prometheus.MustRegister(memory_gc_bytes)
}

type memoryEntry struct {
	data       map[Kind][]byte
	generation int
	size       int64
}

func newMemoryEntry(generation int) memoryEntry {
	return memoryEntry{
		generation: generation,
		data:       make(map[Kind][]byte),
		size:       0,
	}
}

func memoryEntryFromLine(generation int, line Line) memoryEntry {
	m := newMemoryEntry(generation)

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
	start := time.Now()
	//m.lock.Lock()
	//defer m.lock.Unlock()

	defer memory_gc_duration.Observe(time.Now().Sub(start).Seconds())
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
		memory_gc_bytes.Add(float64(m.data[oldestKey].size))
		delete(m.data, oldestKey)
	}
}

func (c *Memory) Put(ns string, uuidAndHash []byte, data Line) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.generation++

	line := memoryEntryFromLine(c.generation, data)

	c.collectGarbage(line.size)

	c.data[ns+string(uuidAndHash)] = line
	c.size += line.size

	return nil
}

func (c *Memory) Get(ns string, kind Kind, uuidAndHash []byte) (int64, io.ReadCloser, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	line, ok := c.data[ns+string(uuidAndHash)]

	if !ok {
		return 0, nil, nil
	}

	if data, ok := line.data[kind]; ok {
		c.generation++
		line.generation = c.generation

		return int64(len(data)), ioutil.NopCloser(bytes.NewReader(data)), nil
	}

	return 0, nil, nil
}

func (m *Memory) PutTransaction(ns string, uuidAndHash []byte) Transaction {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return &MemoryTx{
		mem:         m,
		ns:          ns,
		uuidAndHash: uuidAndHash,
		entry:       newMemoryEntry(m.generation),
	}
}

type MemoryTx struct {
	mem         *Memory
	ns          string
	uuidAndHash []byte
	entry       memoryEntry
}

func (t *MemoryTx) Put(size int64, kind Kind, r io.Reader) error {
	data := make([]byte, size)
	_, err := io.ReadFull(r, data)
	if err != nil {
		return err
	}
	t.entry.data[kind] = data
	t.entry.size += size
	return nil
}

func (t *MemoryTx) Commit() error {
	t.mem.lock.Lock()
	defer t.mem.lock.Unlock()
	t.mem.generation++

	t.entry.generation = t.mem.generation

	t.mem.collectGarbage(t.entry.size)

	t.mem.data[t.ns+string(t.uuidAndHash)] = t.entry
	t.mem.size += t.entry.size

	return nil
}

func (t *MemoryTx) Abort() error {
	// NOP - GC will remove the reference.
	return nil
}
