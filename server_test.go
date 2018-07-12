package ucs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"testing"

	"gitlab.com/msiebuhr/ucs/cache"
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
	expected := fmt.Sprintf("%08x+i%016x%016s%016s", 0xfe, len(data), "dead", "beef") + string(data)
	if !bytes.Equal(out, []byte(expected)) {
		t.Errorf("Expected reply for request to be\n `%s`, got\n `%s`", expected, string(out))
	}
}

func TestCacheMultiPutAndGet(t *testing.T) {
	client, server := net.Pipe()
	s := NewServer()
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
	expected := fmt.Sprintf("%08x+i%016x%016s%016s%s+a%016x%016s%016s%s", 0xfe, len(data), "dead", "beef", data, len(data), "dead", "beef", data)
	if !bytes.Equal(out, []byte(expected)) {
		t.Errorf("Expected reply for request to be\n `%s`, got\n `%s`", expected, string(out))
	}
}

func TestWrongCmdType(t *testing.T) {
	client, server := net.Pipe()
	s := NewServer()
	go s.handleRequest(context.Background(), server)

	go func() {
		client.Write([]byte("000000fepX0000000000000001x"))
	}()

	out, err := ioutil.ReadAll(client)
	if err != nil {
		t.Errorf("Error reading response: %s", err)
	}
	expected := fmt.Sprintf("%08x", 0xfe)
	if !bytes.Equal(out, []byte(expected)) {
		t.Errorf("Expected reply for request to be\n `%s`, got\n `%s`", expected, string(out))
	}
}

// Quick benchmarking
func BenchmarkMemory1mb(b *testing.B) {
	s := NewServer(
	//func (l *Server) {l.Log = log.New(os.Stdout, "", 0)}
	)

	HelpBenchmarkServerGets(b, s, 1024*1024)
}

func BenchmarkFS1mb(b *testing.B) {
	c, err := cache.NewFS(func(f *cache.FS) { f.Basepath = "./testdata" })
	if err != nil {
		b.Fatalf("Error creating FS: %s", err)
	}

	defer func() {
		os.RemoveAll(c.Basepath)
	}()

	s := NewServer(
		//func (l *Server) {l.Log = log.New(os.Stdout, "", 0)}
		func(s *Server) { s.Cache = c },
	)

	HelpBenchmarkServerGets(b, s, 1024*1024)
}

func HelpBenchmarkServerGets(b *testing.B, s *Server, size int64) {
	client, server := net.Pipe()
	go s.handleRequest(context.Background(), server)

	// Handshake
	fmt.Fprintf(client, "%08x", 0xfe)
	io.CopyN(ioutil.Discard, client, 8)

	// Put stuff
	data := make([]byte, size)
	rand.Read(data)
	b.SetBytes(size)
	fmt.Fprintf(client, "ts%016s%016s", "dead", "beef")
	fmt.Fprintf(client, "pi%016x", size)
	client.Write(data)
	fmt.Fprintf(client, "te")

	b.ResetTimer()

	for i := 0; i < b.N; i += 1 {
		fmt.Fprintf(client, "gi%016s%016s", "dead", "beef")
		io.CopyN(ioutil.Discard, client, 2+16+32+size)
	}

	client.Write([]byte("q"))
}
