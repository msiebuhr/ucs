package cache

import (
	"os"
	"time"
	"syscall"
)

// Get A-time from a given os.Fileinfo
func fileinfo_atime(fi os.FileInfo) time.Time {
	if stat_t, ok := fi.Sys().(*syscall.Stat_t); ok {
		secs, nsec := stat_t.Atimespec.Unix()
		return time.Unix(secs, nsec)
	}

	// Fallback: last modification time
	return fi.ModTime()
}
