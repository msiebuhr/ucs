package ucs

import (
	//"bytes"
	//"errors"
	"fmt"
	"io"
	"math/rand"
)

// Returns a reader that generates size bytes based on the given seed.
func NewRandReader(seed, size int64) io.Reader {
	source := rand.NewSource(seed)
	reader := rand.New(source)
	return io.LimitReader(reader, size)
}

// Writer that expectes output from RandReader.
type RandWriter struct {
	// TODO: Make sure we can track if we're given too few bytes...
	Reader    io.Reader
	err       error
	size      int64
	bytesRead int64
}

func NewRandWriter(seed, size int64) *RandWriter {
	return &RandWriter{
		Reader:    NewRandReader(seed, size),
		err:       nil,
		size:      size,
		bytesRead: 0,
	}
}

func (rw *RandWriter) Write(p []byte) (n int, err error) {
	// Bail early if we have already failed elsewhere
	if rw.err != nil {
		return len(p), nil
	}

	// Read same stuff from internal reader
	truth := make([]byte, len(p))
	rw.Reader.Read(truth)

	// Compare data byte-for-byte and check for errors
	for i := 0; i < len(p); i += 1 {
		// TODO: Skip if we already have bytes
		if p[i] != truth[i] && rw.err == nil {
			rw.err = fmt.Errorf("Different data detected at byte %d", rw.bytesRead)
			//return i, nil;
		}
		rw.bytesRead += 1
	}

	return len(p), nil
}

func (rw RandWriter) Done() error {
	if rw.err != nil {
		return rw.err
	}

	if rw.size != rw.bytesRead {
		return fmt.Errorf("Expected %d bytes, got %d", rw.size, rw.bytesRead)
	}

	return nil
}
