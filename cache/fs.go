package cache

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	fs_gc_duration = prometheus.NewSummary(prometheus.SummaryOpts{
		Name: "ucs_fscache_gc_duration_seconds",
		Help: "Time spent deleting data",
	})
	fs_gc_bytes = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "ucs_fscache_gc_removed_bytes",
		Help: "Bytes deleted by GC",
	})
	fs_size = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ucs_fscache_size_bytes",
		Help: "Size of cache in bytes",
	}, []string{"namespace"})
	fs_quota = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ucs_fscache_quota_bytes",
		Help: "Size of quota in bytes",
	})
)

func init() {
	prometheus.MustRegister(fs_gc_duration)
	prometheus.MustRegister(fs_gc_bytes)
	prometheus.MustRegister(fs_size)
	prometheus.MustRegister(fs_quota)
}

type FS struct {
	lock     sync.RWMutex
	Basepath string
	Size     int64
	Quota    int64
}

func NewFS(options ...func(*FS)) (*FS, error) {
	fs := &FS{Basepath: "./unity-cache"}
	for _, f := range options {
		f(fs)
	}

	// Make sure FS is an absolute path
	path, err := filepath.Abs(fs.Basepath)
	if err != nil {
		return fs, err
	}
	fs.Basepath = path

	// Kick off GC so we can get proper sizing info
	go fs.collectGarbage()

	return fs, nil
}

// Remove old files (as measured by the most recent ATIME of any entry sharing
// the same UUID/hash)
// Also re-calculates the total size of cache directory, now we're at scanning
// everything anyway...
func (fs *FS) collectGarbage() {
	// Report quota up front
	fs_quota.Set(float64(fs.Quota))

	start := time.Now()

	fs.lock.Lock()

	defer fs_gc_duration.Observe(time.Now().Sub(start).Seconds())
	defer fs.lock.Unlock()

	fs.Size = 0

	// Find all namespaces
	dir, err := os.Open(fs.Basepath)
	if err != nil {
		//fs.Log.Printf("GC error: %s", err)
		return
	}

	entries, err := dir.Readdir(0)
	if err != nil {
		//fs.log.Printf("GC error: %s", err)
		return
	}

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
			// TODO: Split into ~256 go-routines for speed
			for i := 0; i < 256; i += 1 {
				dirname := filepath.Join(fs.Basepath, ns, fmt.Sprintf("%02x", i))
				dir, err := os.Open(dirname)
				if err != nil {
					continue
				}
				entries, err := dir.Readdir(0)
				if err != nil {
					continue
				}

				for _, entry := range entries {
					if entry.IsDir() {
						continue
					}

					// Count up sizes of everything
					//fs.Size += entry.Size()
					sizes[nsIndex] += entry.Size()

					// Check if it's the oldest thing we've found in this directory
					t := fileinfo_atime(entry)
					oldIndex := i + 256*nsIndex
					if len(old[oldIndex].uuidAndHash) == 0 || t.Before(old[i].time) {
						old[oldIndex].ns = ns
						old[oldIndex].uuidAndHash = entry.Name() // TODO: Chop off extension
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
	fs.Size = 0
	for _, size := range sizes {
		fs.Size += size
	}

	sort.Sort(old)

	// Ideally, we should delete the very oldest stuff first (and both info and
	// asset/resource), and then re-scan that directory.
	// But I'm lazy right now - let's just delete the oldst thing we found in
	// all folders and see how far that get's us.
	for i := 0; i < len(old) && fs.Size > fs.Quota; i += 1 {
		if old[i].uuidAndHash == "" {
			continue
		}

		path := filepath.Join(fs.Basepath, old[i].ns, fmt.Sprintf("%02x", i), old[i].uuidAndHash)

		err := os.Remove(path)
		if err == nil {
			fs_gc_bytes.Add(float64(old[i].size))
			fs.Size -= old[i].size
		}

		// Bail if we get below the quota
		if fs.Size <= fs.Quota {
			return
		}
	}

	// If we're still over quota, do another round of GC'ing
	if fs.Size > fs.Quota {
		go fs.collectGarbage()
	}
}

func (fs *FS) generatePath(ns string, kind Kind, uuidAndHash []byte) string {
	var suffix string
	switch kind {
	case KIND_ASSET:
		suffix = "bin"
	case KIND_INFO:
		suffix = "info"
	case KIND_RESOURCE:
		suffix = "resource"
	default:
		suffix = ".UNKNOWN_TYPE"
	}

	if ns == "" {
		ns = "__default"
	}

	return filepath.Join(fs.Basepath, ns, fmt.Sprintf("%02x", uuidAndHash[:1]), fmt.Sprintf("%016x-%016x.%s", uuidAndHash[:16], uuidAndHash[16:], suffix))
}

func (fs *FS) putKind(ns string, kind Kind, uuidAndHash, data []byte) error {
	path := fs.generatePath(ns, kind, uuidAndHash)

	//fs.lock.Lock()
	//defer fs.lock.Unlock()

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	f.Write(data)

	return nil
}

func (fs *FS) Put(ns string, uuidAndHash []byte, data Line) error {
	fs.lock.Lock()
	defer fs.lock.Unlock()

	// Kick of GC if we're above the size
	fs.Size += data.Size()
	fs_size.WithLabelValues(ns).Add(float64(data.Size()))
	if fs.Size > fs.Quota {
		go fs.collectGarbage()
	}

	// Make sure leading directory exists!
	leadingPath := filepath.Join(fs.Basepath, ns, fmt.Sprintf("%02x", uuidAndHash[:1]))
	os.MkdirAll(leadingPath, os.ModePerm)

	// Loop over types in the Put
	if data.Info != nil {
		err := fs.putKind(ns, KIND_INFO, uuidAndHash, *data.Info)
		if err != nil {
			return err
		}
	}
	if data.Resource != nil {
		err := fs.putKind(ns, KIND_RESOURCE, uuidAndHash, *data.Resource)
		if err != nil {
			return err
		}
	}
	if data.Asset != nil {
		err := fs.putKind(ns, KIND_ASSET, uuidAndHash, *data.Asset)
		if err != nil {
			return err
		}
	}

	return nil
}

func (fs *FS) Get(ns string, kind Kind, uuidAndHash []byte) (int64, io.ReadCloser, error) {
	path := fs.generatePath(ns, kind, uuidAndHash)

	fs.lock.RLock()
	defer fs.lock.RUnlock()

	f, err := os.Open(path)
	if err != nil && os.IsNotExist(err) {
		return 0, nil, nil
	} else if err != nil {
		return 0, nil, err
	}

	stat, err := f.Stat()
	if err != nil {
		return 0, nil, err
	}

	return stat.Size(), f, nil
}

func (fs *FS) PutTransaction(ns string, uuidAndHash []byte) Transaction {
	return &FSTx{
		fs:          fs,
		ns:          ns,
		nsPrefix:    "should-be-replaced",
		uuidAndHash: uuidAndHash,
	}
}

type FSTx struct {
	fs          *FS
	ns          string
	nsPrefix    string
	uuidAndHash []byte
}

func (t *FSTx) Put(size int64, kind Kind, r io.Reader) error {
	path := t.fs.generatePath(t.nsPrefix+t.ns, kind, t.uuidAndHash)

	// Make sure leading directory exists!
	leadingPath := filepath.Join(t.fs.Basepath, t.nsPrefix+t.ns, fmt.Sprintf("%02x", t.uuidAndHash[:1]))
	os.MkdirAll(leadingPath, os.ModePerm)

	// TODO: Ensure paths are prefixed with t.fs.Basepath (i.e. if nsPrefix has dots in it...)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, r)
	return err
}

func (t *FSTx) Commit() error {
	// TODO: We need to move individual files, as the new folder could already exist (and OS's usually aren't nice and does merging of directory contents for us).
	newpath := filepath.Join(
		t.fs.Basepath,
		t.ns,
		fmt.Sprintf("%02x", t.uuidAndHash[:1]),
	)
	os.MkdirAll(newpath, os.ModePerm)

	var err error

	for _, k := range []Kind{KIND_ASSET, KIND_INFO, KIND_RESOURCE} {
		from := t.fs.generatePath(t.nsPrefix+t.ns, k, t.uuidAndHash)
		to := t.fs.generatePath(t.ns, k, t.uuidAndHash)

		e := os.Rename(from, to)
		if e != nil {
			err = e
		}
	}
	return err
}

func (t *FSTx) Abort() error {
	path := filepath.Join(
		t.fs.Basepath,
		t.nsPrefix+t.ns,
		fmt.Sprintf("%02x", t.uuidAndHash[:1]),
	)

	// TODO: Ensure path is prefixed with t.fs.Basepath (i.e. if nsPrefix has dots in it...)

	return os.RemoveAll(path)
}
