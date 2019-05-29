package ucs

import (
	"fmt"
	"io"
	"strings"
)

// PUT Objects. A plain reader, but we need a size up-front.
// TODO: Implement lot's of magic (file sizes, string.NewReader) etc. This
// should eventually be put in PutRequest as internal helper functions.
type PutObject struct {
	r    io.Reader
	size int
}

func NewPutObject(r io.Reader, size int) *PutObject {
	return &PutObject{
		r:    r,
		size: size,
	}
}

// PUT string wrapper
func PutString(s string) *PutObject {
	return NewPutObject(
		strings.NewReader(s),
		len([]byte(s)),
	)
}

type PutRequest struct {
	io.Reader
	uuidAndHash []byte
	info        *PutObject
	asset       *PutObject
	resource    *PutObject
}

func Put(uuidAndHash []byte, i *PutObject, a *PutObject, r *PutObject) *PutRequest {
	return &PutRequest{
		uuidAndHash: uuidAndHash,
		info:        i,
		asset:       a,
		resource:    r,
	}
}

func (p PutRequest) ReadResponse(r io.Reader) error {
	return nil
}

func (p PutRequest) WriteTo(w io.Writer) (int64, error) {
	var written int64 = 0
	n, err := fmt.Fprintf(w, "ts%s", p.uuidAndHash)
	written += int64(n)
	if err != nil {
		return written, err
	}

	if p.info != nil {
		fmt.Fprintf(w, "pi%016x", p.info.size)
		n64, err := io.Copy(w, p.info.r)
		written += n64
		if err != nil {
			return written, err
		}
		// TODO: Err handling
	}

	if p.asset != nil {
		fmt.Fprintf(w, "pa%016x", p.asset.size)
		n64, err := io.Copy(w, p.asset.r)
		written += n64
		if err != nil {
			return written, err
		}
		// TODO: Err handling
	}

	if p.resource != nil {
		fmt.Fprintf(w, "pr%016x", p.resource.size)
		n64, err := io.Copy(w, p.resource.r)
		written += n64
		if err != nil {
			return written, err
		}
		// TODO: Err handling
	}

	n, err = fmt.Fprintf(w, "te")
	written += int64(n)
	return written, err
}
