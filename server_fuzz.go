package ucs

import (
	"context"
	"net"
)

func Fuzz(data []byte) int {
	client, server := net.Pipe()
	s := NewServer()

	go func() {
		client.Write(data)
	}()

	s.handleRequest(context.Background(), server)
	return 0
}
