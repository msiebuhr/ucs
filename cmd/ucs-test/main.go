package main

import (
	"flag"
	"io"
	"log"
	"math/rand"
	"net"

	"github.com/msiebuhr/ucs"
	"github.com/msiebuhr/ucs/cache"
)

var (
	size int64
	seed int64
)

func init() {
	flag.Int64Var(&size, "size", 1e9, "Bytes to upload")
	flag.Int64Var(&seed, "seed", 1e9, "Random generator seed")
}

func main() {
	flag.Parse()

	rand.Seed(seed)

	conn, err := net.Dial("tcp", ":8126")
	if err != nil {
		log.Fatalf("Could not connect: %s", err)
	}
	defer conn.Close()

	c := ucs.NewBulkClientConn(conn)

	// Send version code and read answer
	version, err := c.NegotiateVersion(0xfe)
	if err != nil {
		log.Fatalf("Could not negotiate version: %v", err)
		return
	}
	log.Printf("Got version %d", version)

	c.Callback = func(k cache.Kind, uuidAndHash []byte, hit bool, data io.Reader) {
		log.Printf("Got response from server %t %x", hit, uuidAndHash)
	}

	// Make a random (failing) request
	randomGuidAndHash := make([]byte, 32)
	rand.Read(randomGuidAndHash)

	c.Get(cache.KIND_INFO, randomGuidAndHash)
	c.Execute()

	log.Printf("Uploading")

	data := make([]byte, size)
	putReq := ucs.Put(
		randomGuidAndHash,
		ucs.PutString(string(data)), nil, nil,
		//PutString(string(data)),
		//PutString(string(data)),
	)
	c.Put(putReq)
	c.Execute()
	log.Printf("Uploading done")

	log.Printf("Fetching again")
	// Try getting it again
	c.Get(cache.KIND_INFO, randomGuidAndHash)
	c.Execute()
	log.Printf("Fetched again")

	c.Quit()
}
