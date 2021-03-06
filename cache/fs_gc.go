package cache

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Internal structure to track old files, sort them by age and delete "sets"
// of them.

type fsCacheEntry struct {
	ns          string
	uuidAndHash []byte
	size        int64
	time        time.Time
}

type fsCacheEntries []fsCacheEntry

// Implement sort.Interface
func (f fsCacheEntries) Len() int { return len(f) }
func (f fsCacheEntries) Less(i, j int) bool {
	// Fiddle a bit with zero-times, so they always come out last
	if f[i].time.IsZero() {
		return false
	}
	if f[j].time.IsZero() {
		return true
	}
	return f[i].time.Before(f[j].time)
}
func (f fsCacheEntries) Swap(i, j int) { f[i], f[j] = f[j], f[i] }

// Parses the uuidAndHash from a base filename, such as the one made by
// `generateFilename()`.
func parseFilename(baseFilename string) ([]byte, error) {
	if len(baseFilename) < 65 {
		return []byte{}, fmt.Errorf("Filename too short")
	}
	// Parse out uuidAndHash
	// TODO(deal with errors);
	first, err := hex.DecodeString(baseFilename[0:32])
	if err != nil {
		return []byte{}, err
	}

	second, err := hex.DecodeString(baseFilename[33:65]) // off-by-one to skip dash in the middle
	if err != nil {
		return []byte{}, err
	}

	return append(first, second...), nil
}

// Find an approximate set of oldest files.
//
// Currently, it does a single pass over all sub-direcotries and picks the
// oldest file from each
func findApproximateOldFiles(basepath string) (int64, fsCacheEntries, error) {
	// Find all namespaces
	dir, err := os.Open(basepath)
	// If the path doesn't exist, we're done GC'ing
	if errors.Is(err, os.ErrNotExist) {
		return 0, fsCacheEntries{}, nil
	} else if err != nil {
		return 0, fsCacheEntries{}, fmt.Errorf("GC Error: %w", err)
	}
	defer dir.Close()

	entries, err := dir.Readdir(0)
	if err != nil {
		return 0, fsCacheEntries{}, err
	}

	// Each namespace has 256 subdirectories
	old := make(fsCacheEntries, 256*len(entries))

	sizes := make([]int64, len(entries))
	allDone := sync.WaitGroup{}

	for nsIndex, ns := range entries {
		if !ns.IsDir() {
			continue
		}
		allDone.Add(1)
		go func(ns string, nsIndex int) {
			defer allDone.Done()
			// There be 256 folders - let's find the oldest one + it's size
			// TODO: Split into ~256 go-routines for speed?
			for i := 0; i < 256; i += 1 {
				dirname := filepath.Join(basepath, ns, fmt.Sprintf("%02x", i))
				dir, err := os.Open(dirname)
				if err != nil {
					continue
				}
				entries, err := dir.Readdir(0)
				if err != nil {
					continue
				}

				oldIndex := i + 256*nsIndex
				for _, entry := range entries {
					if entry.IsDir() {
						continue
					}

					// Count up sizes of everything
					//fs.Size += entry.Size()
					sizes[nsIndex] += entry.Size()

					// Check if it's the oldest thing we've found in this directory
					t := fileinfo_atime(entry)
					uuidAndHash, err := parseFilename(entry.Name())
					if err != nil {
						continue
					}

					// Same resource, just different kind.

					// FIXME: We're not guaranteed that the items will come in
					// alphabetical order (which would make this optimization
					// 100% foolproof), but I think we're pretty close with all
					// files of the same uuidAndHash having roughly same
					// timestamp, so we'll ignore all the others...
					if bytes.Equal(old[oldIndex].uuidAndHash, uuidAndHash) {
						old[oldIndex].size += entry.Size()
						if old[oldIndex].time.Before(t) {
							old[oldIndex].time = t
						}
					}

					if len(old[oldIndex].uuidAndHash) == 0 || t.Before(old[oldIndex].time) {
						old[oldIndex].ns = ns
						old[oldIndex].uuidAndHash = uuidAndHash
						old[oldIndex].size = entry.Size()
						old[oldIndex].time = t
					}
				}
				dir.Close()
			}
			fs_size.WithLabelValues(ns).Set(float64(sizes[nsIndex]))
		}(ns.Name(), nsIndex)
	}
	dir.Close()
	allDone.Wait()

	// Add up sizes
	var totalSize int64 = 0
	for _, size := range sizes {
		totalSize += size
	}

	// Return only non-empty objecs
	sort.Sort(old)

	// TODO: We could do this bookkeeping in the main loop...
	found_elements := 0
	for i, elem := range old {
		if elem.time.IsZero() {
			found_elements = i
			break
		}
	}

	return totalSize, old[0:found_elements], nil
}
