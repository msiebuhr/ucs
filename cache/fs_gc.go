package cache

import (
	"time"
)

// Internal structure to track old files, sort them by age and delete "sets"
// of them.

type fsCacheEntry struct {
	ns          string
	uuidAndHash string
	size        int64
	time        time.Time
}

type fsCacheEntries []fsCacheEntry

// Implement sort.Interface
func (f fsCacheEntries) Len() int           { return len(f) }
func (f fsCacheEntries) Less(i, j int) bool { return f[i].time.Before(f[j].time) }
func (f fsCacheEntries) Swap(i, j int)      { f[i], f[j] = f[j], f[i] }
