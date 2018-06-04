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
	"time"
)

func PrettyUuidAndHash(d []byte) string {
	return fmt.Sprintf("%x/%x", d[:16], d[17:])
}

const (
	CONN_TYPE = "tcp"
	CONN_PORT = ":8126"
)

type Server struct {
	Cache *CacheMemory
}

// Set up a new server
func NewServer(options ...func(*Server)) *Server {
	s := &Server{Cache: NewCacheMemory()}

	for _, f := range options {
		f(s)
	}

	return s
}

// Listen for commands
func (s *Server) Listen(ctx context.Context) {
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
		go s.handleRequest(ctx, conn)
	}
}

func readVersionNumber(rw *bufio.ReadWriter) (uint32, error) {
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

// Handles incoming requests.
func (s *Server) handleRequest(ctx context.Context, conn net.Conn) {
	defer func() {
		log.Println("Closing connection")
		conn.Close()
	}()
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	defer rw.Flush()

	// Set deadline for getting data five seconds in the future
	conn.SetDeadline(time.Now().Add(30 * time.Second))

	trx := make([]byte, 0)
	trxData := CacheLine{}

	// First, read uint32 version number
	version, err := readVersionNumber(rw)
	if err != nil {
		log.Printf("Could not read client version: %s", err)
		return
	}
	ctx = context.WithValue(ctx, "version", version)

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
		// Extend Read-dealine by five seconds for each command we process
		conn.SetDeadline(time.Now().Add(30 * time.Second))

		// Flush version or previous command
		// The original server get really confused if clients do agressive
		// streaming. Explicitly waiting for output from previous command
		// seem to make it happy...
		rw.Flush()

		cmd, err := rw.ReadByte()
		if err == io.EOF {
			log.Println("Client hangup; Quitting")
			return
		} else if err != nil {
			log.Println("Error reading command:", err)
			return
		}

		// Quit command
		if cmd == 'q' {
			log.Printf("Got command %c; Quitting", cmd)
			return
		}

		// Get type
		cmdType, err := rw.ReadByte()
		if err != nil {
			log.Println("Error reading command type:", err)
			return
		}
		log.Printf("Got command %c/%c", cmd, cmdType)

		// GET
		if cmd == 'g' {
			// Read uuidAndHash
			uuidAndHash := make([]byte, 32)
			_, err := io.ReadFull(rw, uuidAndHash)

			log.Printf("Get / %c %s", cmdType, PrettyUuidAndHash(uuidAndHash))

			ok, err := s.Cache.Has(Kind(cmdType), uuidAndHash)
			if err != nil {
				log.Println("Error reading from cache:", err)
				fmt.Fprintf(rw, "-%c%s", cmdType, uuidAndHash)
				continue
			}
			if !ok {
				log.Println("Cache miss")
				fmt.Fprintf(rw, "-%c%s", cmdType, uuidAndHash)
				continue
			}

			data, err := s.Cache.Get(Kind(cmdType), uuidAndHash)
			if err != nil {
				log.Println("Error reading from cache:", err)
				fmt.Fprintf(rw, "-%c%s", cmdType, uuidAndHash)
				continue
			}
			fmt.Fprintf(rw, "+%c%016x%s", cmdType, len(data), uuidAndHash)
			rw.Write(data)
			continue
		}

		// Transaction start
		if cmd == 't' && cmdType == 's' {
			// Bail if we're already in a command
			if len(trx) > 0 {
				log.Println("Error starting trx inside trx")
				return
			}

			// Read uuidAndHash
			uuidAndHash := make([]byte, 32)
			_, err := io.ReadFull(rw, uuidAndHash)
			if err != nil {
				log.Println("Error reading uuid+hash:", err)
				return
			}

			log.Printf("Transaction started for %s", PrettyUuidAndHash(uuidAndHash))

			trx = uuidAndHash
			continue
		}

		// Transaction end
		if cmd == 't' && cmdType == 'e' {
			if len(trx) == 0 {
				log.Println("Error ending trx - none started")
				return
			}

			err := s.Cache.Put(trx, trxData)
			if err != nil {
				log.Println("Error ending trx - cache put error:", err)
				continue
			}

			trx = []byte{}
			trxData = CacheLine{}
			continue
		}

		// Put
		if cmd == 'p' {
			// Read size
			sizeBytes := make([]byte, 16)
			_, err := io.ReadFull(rw, sizeBytes)
			if err != nil {
				log.Println("Error putting - cannot read size:", err)
				return
			}

			// Parse size
			size, err := strconv.ParseUint(string(sizeBytes), 16, 64)
			if err != nil {
				log.Printf("Error putting - cannot parse size '%x': %s", sizeBytes, err)
				return
			}
			log.Println("Put, size", string(sizeBytes), size)

			// TODO: Cache should probably have the reader embedded
			trxData.PutReader(Kind(cmdType), size, rw)
			continue
		}

		// Invalid command
		log.Printf("Invalid command: %c%c", cmd, cmdType)
		return
	}
}
