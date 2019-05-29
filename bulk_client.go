package ucs

import (
	//"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/msiebuhr/ucs/cache"
)

// BulkClient sends requests in bulk, like Unity does. That is, it sends
// *all* get-requests before it begins reading responses.
type BulkClient struct {
	Conn     io.ReadWriteCloser
	Callback func(K cache.Kind, uuidAndHash []byte, hit bool, data io.Reader)

	getRequests []io.Reader
	putRequests []io.WriterTo
}

func NewBulkClientConn(conn io.ReadWriteCloser) *BulkClient {
	return &BulkClient{Conn: conn}
}

func (c BulkClient) NegotiateVersion(my uint32) (uint32, error) {
	fmt.Fprintf(c.Conn, "%08x", my)
	versionBytes := make([]byte, 8)
	_, err := io.ReadFull(c.Conn, versionBytes)
	if err != nil {
		return 0, err
	}

	version, err := strconv.ParseUint(string(versionBytes), 16, 32)
	return uint32(version), err
}

// Gracefully quit the current connection and close down
func (c BulkClient) Quit() error {
	_, err := fmt.Fprintf(c.Conn, "q")
	return err
}

// Close the connection. Unpolite, I guess, but that's what Unity is
// observed to do in the wild.
func (c BulkClient) Close() error {
	return c.Conn.Close()
}

// Enqueue a get-request and wait for response to show up
func (c *BulkClient) Get(K cache.Kind, uuidAndHash []byte) error {
	command := fmt.Sprintf("g%c%s", K, uuidAndHash)
	c.getRequests = append(c.getRequests, strings.NewReader(command))
	return nil
}

// Callback that will get replies for get-requests
func (c *BulkClient) GetCallback(callback func(K cache.Kind, uuidAndHash []byte, hit bool, data io.Reader)) {
	c.Callback = callback
}

// Putting data
func (c *BulkClient) Put(p *PutRequest) {
	c.putRequests = append(c.putRequests, p)
}

func (c *BulkClient) Execute() error {
	defer func() {
		c.Conn.Close()
	}()

	// We should send enqueued requests here
	for _, cmd := range c.getRequests {
		_, err := io.Copy(c.Conn, cmd)
		if err != nil {
			return err
		}
	}

	// TODO: When does it send put-requests? First, after-get or after-fetch?
	for _, cmd := range c.putRequests {
		_, err := cmd.WriteTo(c.Conn)
		if err != nil {
			return err
		}
	}

	// Read a response for each PUT-request we did
	// TODO: Read only for PUTs
	for i := 0; i < len(c.getRequests); i += 1 {
		// Positive or negative response
		typeAndHit := make([]byte, 2)
		_, err := io.ReadFull(c.Conn, typeAndHit)
		// New command, but is connection closed
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		// What to do now?
		//if typeAndHit[1] != byte(g.K) {
		//	return errors.New("Unexpected kind returned")
		//}

		uuidAndHash := make([]byte, 32)
		var size uint64

		if typeAndHit[0] == '+' {
			sizeBytes := make([]byte, 16)
			_, err := io.ReadFull(c.Conn, sizeBytes)
			if err != nil {
				return err
			}
			size, err = strconv.ParseUint(string(sizeBytes), 16, 64)
			if err != nil {
				return err
			}
		}
		_, err = io.ReadFull(c.Conn, uuidAndHash)

		hit := true
		if typeAndHit[0] == '-' {
			hit = false
			size = 0
		}

		// Callback to let the client handle returned data
		c.Callback(
			cache.Kind(typeAndHit[1]),
			uuidAndHash,
			hit,
			io.LimitReader(c.Conn, int64(size)),
		)
	}

	// Clear queue?
	c.getRequests = []io.Reader{}

	return nil
}
