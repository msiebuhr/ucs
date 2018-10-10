package main

import (
	"flag"
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

	data := make([]byte, size)
	putReq := ucs.Put(
		randomGuidAndHash,
		ucs.PutString(string(data)), nil, nil,
		//PutString(string(data)),
		//PutString(string(data)),
	)
	c.Execute(putReq)

	// Try getting it again
	req = ucs.Get(cache.KIND_INFO, randomGuidAndHash)
	go func() {
		data, err := ioutil.ReadAll(req)
		log.Printf("Got response from client %t %d, %v", req.Hit(), len(data), err)
	}()
	c.Execute(req)

	c.Quit()
}
