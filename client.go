package ucs

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/msiebuhr/ucs/cache"
)

// Cache requester is for talking to the cache
type CacheRequester interface {
	// Write out what should be sent as the request
	WriteTo(io.Writer) (int64, error)
	// Read the response of the wire
	// TODO: Use io.ReaderFrom(io.Reader) (int64, error)
	ReadResponse(io.Reader) error
}

// Get request returns the cache lookup in it's reader
type GetRequest struct {
	io.Reader
	K           cache.Kind
	uuidAndHash []byte
	r           io.Reader
	w           io.WriteCloser
	hit         chan bool
}

func Get(K cache.Kind, uuidAndHash []byte) *GetRequest {
	r, w := io.Pipe()

	return &GetRequest{
		K:           K,
		uuidAndHash: uuidAndHash,
		r:           r,
		w:           w,
		hit:         make(chan bool, 1),
	}
}

func (g GetRequest) Hit() bool {
	return <-g.hit
}

func (g GetRequest) Read(p []byte) (int, error) {
	return g.r.Read(p)
}

func (g GetRequest) WriteTo(w io.Writer) (int64, error) {
	n, err := fmt.Fprintf(w, "g%s%s", g.K, g.uuidAndHash)
	return int64(n), err
}

func (g GetRequest) ReadResponse(r io.Reader) error {
	// Positive or negative response
	typeAndHit := make([]byte, 2)
	_, err := io.ReadFull(r, typeAndHit)
	if err != nil {
		return err
	}

	// What to do now?
	if typeAndHit[1] != byte(g.K) {
		return errors.New("Unexpected kind returned")
	}

	uuidAndHash := make([]byte, 32)
	var size uint64

	if typeAndHit[0] == '+' {
		g.hit <- true
		sizeBytes := make([]byte, 16)
		_, err := io.ReadFull(r, sizeBytes)
		if err != nil {
			return err
		}
		size, err = strconv.ParseUint(string(sizeBytes), 16, 64)
		if err != nil {
			return err
		}
	}
	_, err = io.ReadFull(r, uuidAndHash)

	if typeAndHit[0] == '-' {
		g.hit <- false
		g.w.Close()
		return nil
	}

	io.CopyN(g.w, r, int64(size))
	g.w.Close()
	return nil
}

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

// Generic cache client
type Client struct {
	Conn io.ReadWriteCloser
}

func NewClient(conn io.ReadWriteCloser) *Client {
	return &Client{Conn: conn}
}

func (c Client) NegotiateVersion(my uint32) (uint32, error) {
	fmt.Fprintf(c.Conn, "%08x", my)
	versionBytes := make([]byte, 8)
	_, err := io.ReadFull(c.Conn, versionBytes)
	if err != nil {
		return 0, err
	}

	version, err := strconv.ParseUint(string(versionBytes), 16, 32)
	return uint32(version), err
}

func (c Client) Execute(req CacheRequester) error {
	_, err := req.WriteTo(c.Conn)
	if err != nil {
		return err
	}

	return req.ReadResponse(c.Conn)
}

// Close the connection. Unpolite, I guess, but that's what Unity is
// observed to do in the wild.
func (c Client) Close() {
	c.Conn.Close()
}

// Gracefully quit the current connection and close down
func (c Client) Quit() {
	c.Conn.Write([]byte{'q'})
	c.Conn.Close()
}
