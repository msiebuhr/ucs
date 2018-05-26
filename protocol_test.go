package ucs

import (
	"bytes"
	"io/ioutil"
	"net"
	"testing"
)

func TestHandshakes(t *testing.T) {
	// Send the regular hex string '00fe'
	client, server := net.Pipe()

	go handleRequest(server)

	client.Write([]byte("000000fe"))
	client.Write([]byte("q"))

	out, err := ioutil.ReadAll(client)
	if err != nil {
		t.Errorf("Error reading response: %s", err)
	}
	if !bytes.Equal(out, []byte("000000fe")) {
		t.Errorf("Expected reply for version `000000fe` to be `000000fe`, got `%s`", out)
	}
}

func TestInvalidVersionHandshake(t *testing.T) {
	client, server := net.Pipe()
	go handleRequest(server)

	client.Write([]byte("000000ff"))

	out, err := ioutil.ReadAll(client)
	if err != nil {
		t.Errorf("Error reading response: %s", err)
	}
	if !bytes.Equal(out, []byte("00000000")) {
		t.Errorf("Expected reply for version `00000000` to be `00000000`, got `%s`", out)
	}
}

func TestShortVersionHandshake(t *testing.T) {
	client, server := net.Pipe()
	go handleRequest(server)

	client.Write([]byte("fe"))
	client.Write([]byte("q"))
	out, err := ioutil.ReadAll(client)
	if err != nil {
		t.Errorf("Error reading response: %s", err)
	}
	if !bytes.Equal(out, []byte("000000fe")) {
		t.Errorf("Expected reply for version `00000000` to be `00000000`, got `%s`", out)
	}
}
