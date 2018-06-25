package ucs

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net"
)

func Fuzz(data []byte) int {
	client, server := net.Pipe()
	s := NewServer()

	// Send each line of data as a packet of it's own
	go func() {
		datas := bytes.Split(data, []byte("\n"))
		for _, d := range datas {
			client.Write(d)
		}
		client.Close()
	}()

	// Read but ignore returned data
	go io.Copy(ioutil.Discard, client)

	s.handleRequest(context.Background(), server)

	return 0
}
