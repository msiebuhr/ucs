package ucs

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"testing"

	"github.com/msiebuhr/ucs/cache"
)

func TestBulkClient(t *testing.T) {
	s := NewServer(func(s *Server) {
		s.Cache = cache.NewMemory(1e7)
		//s.Log = log.New(os.Stdout, "server: ", 0)
	})
	defer s.Stop()

	client, server := net.Pipe()
	go s.handleRequest(context.Background(), server)
	c := NewBulkClientConn(client)

	uuidAndHash := make([]byte, 32)
	rand.Read(uuidAndHash)

	// Negotiate server version
	serverVersion, err := c.NegotiateVersion(0xfe)
	if serverVersion != 0xfe || err != nil {
		t.Errorf("Expected 0xfe && no error from handshake, got %x, %s", serverVersion, err)
	}

	// Upload an asset
	c.Put(Put(uuidAndHash, PutString("information blob"), nil, nil))

	// Callback should throw error if something bad happens
	c.Callback = func(k cache.Kind, uuidAndHash []byte, hit bool, data io.Reader) {
		t.Errorf("Put-request should not do any callbacks")
	}

	err = c.Execute()
	if err != nil {
		t.Errorf("Unexpected error uploading data: %s", err)
	}

	// Set up new client to ask for resources
	client, server = net.Pipe()
	go s.handleRequest(context.Background(), server)
	c = NewBulkClientConn(client)

	// Negotiate server version
	serverVersion, err = c.NegotiateVersion(0xfe)
	if serverVersion != 0xfe || err != nil {
		t.Errorf("Expected 0xfe && no error from handshake, got %x, %s", serverVersion, err)
	}

	// Ask for two assets
	c.Get(cache.KIND_INFO, uuidAndHash)
	c.Get(cache.KIND_ASSET, uuidAndHash)
	c.Get(cache.KIND_RESOURCE, uuidAndHash)
	c.Callback = func(k cache.Kind, uuidAndHashBack []byte, hit bool, reader io.Reader) {
		data, err := ioutil.ReadAll(reader)
		if err != nil {
			t.Errorf("Unexpected error reading data in callback: %s", err)
		}

		if !bytes.Equal(uuidAndHash, uuidAndHashBack) {
			t.Errorf("Unexpected UUID/Hash in callback: %d (only asked for %s", uuidAndHashBack, uuidAndHash)
			return
		}

		// We should get an info-blob back
		if k == cache.KIND_INFO {
			if !hit || !bytes.Equal(data, []byte("information blob")) {
				t.Errorf(
					"Unexpected callback for %c/hit:%t/data:%s",
					k, hit, data,
				)
			}
			return
		}

		// Everything else should be miss'es
		if hit || len(data) > 0 {
			t.Errorf(
				"Unexpected data returned %c/%s: %t / %s",
				k, uuidAndHashBack, hit, data,
			)
		}
	}

	err = c.Execute()
	if err != nil {
		t.Fatalf("Unexpected error Execute()'ing: %s", err)
	}

	// Read responses
}
