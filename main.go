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

type CacheMemory struct {
	data map[string][]byte
}

func NewCacheMemory() *CacheMemory {
	return &CacheMemory{data: make(map[string][]byte)}
}

func (c *CacheMemory) Has(kind byte, uuidAndHash []byte) (bool, error) {
	log.Printf("CacheMemory.Has %c %s", kind, PrettyUuidAndHash(uuidAndHash))
	key := fmt.Sprintf("%x%s", kind, uuidAndHash)

	_, ok := c.data[key]
	return ok, nil
}

func (c *CacheMemory) Put(kind byte, uuidAndHash []byte, data []byte) error {
	log.Printf("CacheMemory.Put %c %s", kind, PrettyUuidAndHash(uuidAndHash))
	key := fmt.Sprintf("%x%s", kind, uuidAndHash)

	c.data[key] = data
	return nil
}

func (c *CacheMemory) Get(kind byte, uuidAndHash []byte) ([]byte, error) {
	log.Printf("CacheMemory.Get %c %s", kind, PrettyUuidAndHash(uuidAndHash))
	key := fmt.Sprintf("%x%s", kind, uuidAndHash)

	if data, ok := c.data[key]; ok {
		return data, nil
	}

	return []byte{}, nil
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

	conn.SetDeadline(time.Now().Add(2 * time.Second))

	cache := NewCacheMemory()
	trx := make([]byte, 0)
	var trxType byte
	var trxData []byte

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
		// Set fresh timeout
		conn.SetDeadline(time.Now().Add(2 * time.Second))

		// Explicitly send back data after each command
		rw.Flush()

		//cmd, err := readTwoByteCommand(rw)
		cmd, err := rw.ReadByte()
		if err != nil {
			log.Println("Error reading command:", err)
			return
		}

		log.Printf("Got command %c", cmd)

		// Quit command
		if cmd == 'q' {
			log.Println("Quitting")
			return
		}

		// Get type
		cmdType, err := rw.ReadByte()
		if err != nil {
			log.Println("Error reading command type:", err)
			return
		}

		log.Printf("Got command type %c", cmdType)

		// GET
		if cmd == 'g' {
			// Read uuidAndHash
			uuidAndHash := make([]byte, 32)
			_, err := io.ReadFull(rw, uuidAndHash)

			log.Printf("Get / %c %s", cmdType, PrettyUuidAndHash(uuidAndHash))

			ok, err := cache.Has(cmdType, uuidAndHash)
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

			data, err := cache.Get(cmdType, uuidAndHash)
			if err != nil {
				log.Println("Error reading from cache:", err)
				fmt.Fprintf(rw, "-%c%s", cmdType, uuidAndHash)
				continue
			}
			fmt.Fprintf(rw, "+%c%08x%s", cmdType, len(data), uuidAndHash)
			rw.Write(data)
			continue
		}

		// Transaction start
		if cmd == 't' && cmdType == 's' {
			log.Printf("Transaction start")
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

			trx = uuidAndHash
		}

		// Transaction end
		if cmd == 't' && cmdType == 'e' {
			log.Printf("Transaction end")
			if len(trx) == 0 {
				log.Println("Error ending trx - none started")
				return
			}

			err := cache.Put(trxType, trx, trxData)
			if err != nil {
				log.Println("Error ending trx - cache put error:", err)
				continue
			}

			trx = []byte{}
			trxType = 0
			trxData = []byte{}
		}

		// Put
		if cmd == 'p' {
			log.Println("PUT")
			// Read size
			sizeBytes := make([]byte, 16)
			_, err := io.ReadFull(rw, sizeBytes)
			if err != nil {
				log.Println("Error putting - cannot read size:", err)
				return
			}
			log.Println("PUT / SIZE BYTES", string(sizeBytes))

			// Parse size
			size, err := strconv.ParseUint(string(sizeBytes), 16, 32)
			if err != nil {
				log.Printf("Error putting - cannot parse size '%x': %s", sizeBytes, err)
				return
			}
			log.Println("PUT / SIZE", size)

			trxData = make([]byte, size)
			_, err = io.ReadFull(rw, trxData)
			if err != nil {
				log.Println("Error putting - cannot read data:", err)
				return
			}

			trxType = cmdType
		}
	}
}
