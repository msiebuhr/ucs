package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"

	"gitlab.com/msiebuhr/ucs"
	"gitlab.com/msiebuhr/ucs/cache"
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

	c := ucs.NewClient(conn)

	// Send version code and read answer
	version, err := c.NegotiateVersion(0xfe)
	if err != nil {
		log.Fatalf("Could not negotiate version: %v", err)
		return
	}
	log.Printf("Got version %d", version)

	// Make a random (failing) request
	randomGuidAndHash := make([]byte, 32)
	rand.Read(randomGuidAndHash)

	req := ucs.Get(cache.KIND_INFO, randomGuidAndHash)
	go func() {
		data, err := ioutil.ReadAll(req)
		log.Printf("Got response from client %t %d, %v", req.Hit(), len(data), err)
	}()
	c.Execute(req)

	// Put something in the cache
	rand.Read(randomGuidAndHash)
	fmt.Fprintf(conn, "ts%s", randomGuidAndHash)
	data := make([]byte, size)
	fmt.Fprintf(conn, "pi%016x%s", len(data), data)
	fmt.Fprintf(conn, "pa%016x%s", len(data), data)
	fmt.Fprintf(conn, "te")

	// Try getting it again
	req = ucs.Get(cache.KIND_INFO, randomGuidAndHash)
	go func() {
		data, err := ioutil.ReadAll(req)
		log.Printf("Got response from client %t %d, %v", req.Hit(), len(data), err)
	}()
	c.Execute(req)

	c.Quit()
}
