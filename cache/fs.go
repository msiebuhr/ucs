package cache

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
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

	transactionCout uint64
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

	// Kick off an initial GC, so we can get proper sizing info
	go fs.collectGarbageOnce()

	return fs, nil
}

func (fs *FS) collectGarbage() {
	var lastSize int64 = -1

	for {
		fs.lock.Lock()
		size := fs.Size
		quota := fs.Quota
		fs.lock.Unlock()

		if size <= quota {
			return
		}
		// Did we make any progress?
		if lastSize == size {
			return
		}

		fs.collectGarbageOnce()
	}
}

// Remove old files (as measured by the most recent ATIME of any entry sharing
// the same UUID/hash)
// Also re-calculates the total size of cache directory, now we're at scanning
// everything anyway...
func (fs *FS) collectGarbageOnce() {
	// Report quota up front
	fs_quota.Set(float64(fs.Quota))

	start := time.Now()

	fs.lock.Lock()

	defer fs_gc_duration.Observe(time.Now().Sub(start).Seconds())
	defer fs.lock.Unlock()

	totalSize, old, err := findApproximateOldFiles(fs.Basepath)

	if err != nil {
		fmt.Printf("Error running GC: %#v\n", err)
		return
	}

	fs.Size = totalSize

	// Ideally, we should delete the very oldest stuff first (and both info and
	// asset/resource), and then re-scan that directory.
	// But I'm lazy right now - let's just delete the oldst thing we found in
	// all folders and see how far that get's us.
	for i := 0; i < len(old) && fs.Size > fs.Quota; i += 1 {
		if len(old[i].uuidAndHash) == 0 {
			continue
		}

		successfulDeletes := 0
		for _, kind := range []Kind{KIND_ASSET, KIND_INFO, KIND_RESOURCE} {
			path := fs.generateFilename(old[i].ns, kind, old[i].uuidAndHash)

			err := os.Remove(path)
			if err != nil {
				successfulDeletes += 1
			}
		}

		// Accounting is approximate, as findApproximateOldFiles() doesn't
		// guarantee that it finds all kinds of a resource in one go (yet we
		// delete them in one go).
		//
		// Next loop of the GC should fix the overall stats, tho.
		if successfulDeletes > 0 {
			size := float64(old[i].size)
			fs_gc_bytes.Add(size)
			fs_size.WithLabelValues(old[i].ns).Sub(size)
			fs.Size -= old[i].size
		}

		// Bail if we get below the quota
		if fs.Size <= fs.Quota {
			return
		}
	}
}

func (fs *FS) generateDir(ns string, uuidAndHash []byte) string {
	if ns == "" {
		ns = "__default"
	}
	return filepath.Join(fs.Basepath, ns, fmt.Sprintf("%02x", uuidAndHash[:1]))
}

func (fs *FS) generateFilename(ns string, kind Kind, uuidAndHash []byte) string {
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

	return filepath.Join(
		fs.generateDir(ns, uuidAndHash),
		fmt.Sprintf("%016x-%016x.%s", uuidAndHash[:16], uuidAndHash[16:], suffix),
	)
}

func (fs *FS) Get(ns string, kind Kind, uuidAndHash []byte) (int64, io.ReadCloser, error) {
	path := fs.generateFilename(ns, kind, uuidAndHash)

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
	count := atomic.AddUint64(&fs.transactionCout, 1)
	return &FSTx{
		fs:          fs,
		ns:          ns,
		nsSuffix:    fmt.Sprintf(".tx-%010d", count),
		uuidAndHash: uuidAndHash,
		kinds:       []Kind{},
	}
}

type FSTx struct {
	fs          *FS
	ns          string
	nsSuffix    string
	uuidAndHash []byte

	// Track what kinds have been uploaded
	kinds []Kind
	size  int64
}

func (t *FSTx) Put(size int64, kind Kind, r io.Reader) error {
	t.kinds = append(t.kinds, kind)
	// Make sure leading directory exists!
	leadingPath := t.fs.generateDir(t.ns, t.uuidAndHash)
	os.MkdirAll(leadingPath, os.ModePerm)

	path := t.fs.generateFilename(t.ns, kind, t.uuidAndHash) + t.nsSuffix

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	t.size += size

	_, err = io.Copy(f, r)
	return err
}

func (t *FSTx) Commit() error {
	for _, k := range t.kinds {
		from := t.fs.generateFilename(t.ns, k, t.uuidAndHash) + t.nsSuffix
		to := t.fs.generateFilename(t.ns, k, t.uuidAndHash)

		err := os.Rename(from, to)
		if err != nil {
			return err
		}
	}

	t.fs.lock.Lock()
	t.fs.Size += t.size
	t.fs.lock.Unlock()
	t.fs.collectGarbage()

	fs_size.WithLabelValues(t.ns).Add(float64(t.size))

	return nil
}

func (t *FSTx) Abort() error {
	for _, k := range t.kinds {
		from := t.fs.generateFilename(t.ns, k, t.uuidAndHash) + t.nsSuffix
		os.Remove(from)
	}
	return nil
}
