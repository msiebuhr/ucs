package ucs

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
)

type Server struct {
}

type Conn struct {
}

const (
	CONN_TYPE = "tcp"
	CONN_PORT = ":8126"
)

// Listen for commands
func Listen(ctx context.Context) {
	l, err := net.Listen(CONN_TYPE, CONN_PORT)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer l.Close()
	fmt.Println("Listening on " + CONN_TYPE + ":" + CONN_PORT)
	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		// Handle connections in a new goroutine.
		go handleRequest(conn)
	}
}

type Handler func(*bufio.ReadWriter)

// Get Asset (ga) command.
// client --- 'ga' (id <128bit GUID><128bit HASH>) --> server
// client <-- '+a' (size <uint64>) (id <128bit GUID><128bit HASH>) + size bytes --- server (found in cache)
// client <-- '-a' (id <128bit GUID><128bit HASH>) --- server (not found in cache)
func handleGA(rw *bufio.ReadWriter) {
	// Read 128 bit GUID
	// READ 128 bit HASH
}

func readUint32Helper(rw *bufio.ReadWriter) (uint32, error) {
	// Peek at some data so we force it to wait for some data so we don't get
	// zero buffer sizes all the time...
	rw.Reader.Peek(2)

	// See how much data is waiting for us.
	bytesToRead := rw.Reader.Buffered()
	// We want to read at most eight hex decimals
	if bytesToRead > 8 {
		bytesToRead = 8
	}
	// And at least two.
	if bytesToRead < 2 {
		bytesToRead = 2
	}

	bytes := make([]byte, bytesToRead)
	_, err := io.ReadFull(rw, bytes)
	if err != nil {
		return 0, err
	}

	// Convert number from hex to a real one
	n, err := strconv.ParseUint(string(bytes), 16, 32)
	return uint32(n), err
}

func readTwoByteCommand(rw *bufio.ReadWriter) (string, error) {
	byteone, err := rw.ReadByte()
	if err != nil {
		return "", err
	}

	// 'q' is a one-byte command, so return that if seen...
	if byteone == 'q' {
		return "q", nil
	}

	bytetwo, err := rw.ReadByte()
	if err != nil {
		return "", err
	}

	return string(byteone) + string(bytetwo), nil
}

// Handles incoming requests.
func handleRequest(conn net.Conn) {
	defer conn.Close()
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	defer rw.Flush()

	// First, read uint32 version number
	version, err := readUint32Helper(rw)
	if err != nil {
		log.Printf("Could not read client version: %s", err)
		return
	}

	// Bail on unknown versions
	if version != 0xfe {
		log.Printf("Got invalid client version %d", version)
		fmt.Fprintf(rw, "%08x", 0)
		return
	}

	// Protocol says to echo version if everything is ok
	log.Printf("Got client version %d", version)
	fmt.Fprintf(rw, "%08x", version)

	for {
		cmd, err := readTwoByteCommand(rw)
		if err != nil {
			log.Println("Error reading command. Got: "+cmd+"\n", err)
			return
		}

		log.Println("Got command", cmd)
		switch cmd {
		case "q":
			log.Println("Quitting")
			return
		case "ga":
			handleGA(rw)
		}
	}
}
