package ucs

import (
	"bytes"
	"fmt"
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

func TestGACacheMiss(t *testing.T) {
	client, server := net.Pipe()
	go handleRequest(server)

	request := fmt.Sprintf("%08xga%032x%032xq", 0xfe, 0x42, 0x42)
	client.Write([]byte(request))

	out, err := ioutil.ReadAll(client)
	if err != nil {
		t.Errorf("Error reading response: %s", err)
	}
	expected := fmt.Sprintf("%08xa-%032x%032x", 0xfe, 0x42, 0x42)
	if !bytes.Equal(out, []byte(expected)) {
		t.Errorf("Expected reply for request `%s` to be\n `%s`, got\n `%s`", request, expected, string(out))
	}
}

func TestGACachePutAndGet(t *testing.T) {
	client, server := net.Pipe()
	go handleRequest(server)

	data := []byte("Here is some very lovely test information for ya'")

	fmt.Fprintf(client, "%08x", 0xfe)
	fmt.Fprintf(client, "ts%032x%032x", 0x42, 0x42)
	fmt.Fprintf(client, "pi%08x", len(data))
	client.Write(data)
	fmt.Fprintf(client, "te")
	fmt.Fprintf(client, "gi%032x%032x", 0x42, 0x42)
	client.Write([]byte("q"))

	out, err := ioutil.ReadAll(client)
	if err != nil {
		t.Errorf("Error reading response: %s", err)
	}
	expected := fmt.Sprintf("%08xi+%08x%032x%032x", 0xfe, len(data), 0x42, 0x42) + string(data)
	if !bytes.Equal(out, []byte(expected)) {
		t.Errorf("Expected reply for request to be\n `%s`, got\n `%s`", expected, string(out))
	}
}
