package ucs

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
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
	var number uint32
	var i uint

	for i = 0; i < 4; i = i + 1 {
		b, err := rw.ReadByte()
		if err != nil {
			return number, err
		}
		number += uint32(b) << (3 - i)
	}

	return number, nil
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
	// Make a buffer to hold incoming data.

	// First, read uint32 version number
	version, err := readUint32Helper(rw)
	log.Println("Got client version", version, err)
	if err != nil {
		rw.Write([]byte("Could not read version"))
		return
	}

	// Reply to command
	// TODO: Protocol says to echo version if everything is ok
	rw.Write([]byte{0, 0, 0, 0})

	for {
		log.Print("Receive command\n")
		cmd, err := readTwoByteCommand(rw)
		if err != nil {
			log.Println("Error reading command. Got: "+cmd+"\n", err)
			return
		}

		log.Println("Got command", cmd)
		switch cmd {
		case "q":
			rw.Write([]byte("BYE"))
			log.Println("Quitting")
			return
		case "ga":
			handleGA(rw)
		}
	}
}
