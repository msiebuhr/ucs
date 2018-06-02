package ucs

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"testing"
)

func TestHandshakes(t *testing.T) {
	// Send the regular hex string '00fe'
	client, server := net.Pipe()
	s := NewServer()
	go s.handleRequest(context.Background(), server)

	go func() {
		client.Write([]byte("000000fe"))
		client.Write([]byte("q"))
	}()

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
	s := NewServer()
	go s.handleRequest(context.Background(), server)

	go func() {
		client.Write([]byte("000000ff"))
	}()

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
	s := NewServer()
	go s.handleRequest(context.Background(), server)

	go func() {
		client.Write([]byte("fe"))
		client.Write([]byte("q"))
	}()

	out, err := ioutil.ReadAll(client)
	if err != nil {
		t.Errorf("Error reading response: %s", err)
	}
	if !bytes.Equal(out, []byte("000000fe")) {
		t.Errorf("Expected reply for version `00000000` to be `00000000`, got `%s`", out)
	}
}

func TestGACacheMiss(t *testing.T) {
	client, server := net.Pipe()
	s := NewServer()
	go s.handleRequest(context.Background(), server)

	request := fmt.Sprintf("%08xga%016s%016sq", 0xfe, "dead", "beef")
	go client.Write([]byte(request))

	out, err := ioutil.ReadAll(client)
	if err != nil {
		t.Errorf("Error reading response: %s", err)
	}
	expected := fmt.Sprintf("%08x-a%016s%016s", 0xfe, "dead", "beef")
	if !bytes.Equal(out, []byte(expected)) {
		t.Errorf("Expected reply for request `%s` to be\n `%s`, got\n `%s`", request, expected, string(out))
	}
}

func TestGACachePutAndGet(t *testing.T) {
	client, server := net.Pipe()
	s := NewServer()
	go s.handleRequest(context.Background(), server)

	data := []byte("Here is some very lovely test information for ya'")

	go func() {
		fmt.Fprintf(client, "%08x", 0xfe)
		fmt.Fprintf(client, "ts%016s%016s", "dead", "beef")
		fmt.Fprintf(client, "pi%016x", len(data))
		client.Write(data)
		fmt.Fprintf(client, "te")
		fmt.Fprintf(client, "gi%016s%016s", "dead", "beef")
		client.Write([]byte("q"))
	}()

	out, err := ioutil.ReadAll(client)
	if err != nil {
		t.Errorf("Error reading response: %s", err)
	}
	expected := fmt.Sprintf("%08x+i%08x%016s%016s", 0xfe, len(data), "dead", "beef") + string(data)
	if !bytes.Equal(out, []byte(expected)) {
		t.Errorf("Expected reply for request to be\n `%s`, got\n `%s`", expected, string(out))
	}
}

func TestCacheMultiPutAndGet(t *testing.T) {
	client, server := net.Pipe()
	s := Server{Cache: NewCacheMemory()}
	go s.handleRequest(context.Background(), server)

	data := []byte("Here is some very lovely test information for ya'")

	go func() {
		fmt.Fprintf(client, "%08x", 0xfe)
		fmt.Fprintf(client, "ts%016s%016s", "dead", "beef")
		fmt.Fprintf(client, "pi%016x", len(data))
		client.Write(data)
		fmt.Fprintf(client, "pa%016x", len(data))
		client.Write(data)
		fmt.Fprintf(client, "te")
		fmt.Fprintf(client, "gi%016s%016s", "dead", "beef")
		fmt.Fprintf(client, "ga%016s%016s", "dead", "beef")
		client.Write([]byte("q"))
	}()

	out, err := ioutil.ReadAll(client)
	if err != nil {
		t.Errorf("Error reading response: %s", err)
	}
	expected := fmt.Sprintf("%08x+i%08x%016s%016s%s+a%08x%016s%016s%s", 0xfe, len(data), "dead", "beef", data, len(data), "dead", "beef", data)
	if !bytes.Equal(out, []byte(expected)) {
		t.Errorf("Expected reply for request to be\n `%s`, got\n `%s`", expected, string(out))
	}
}
