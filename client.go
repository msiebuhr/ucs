package ucs

import (
	"fmt"
	"io"
	"strconv"
)

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
