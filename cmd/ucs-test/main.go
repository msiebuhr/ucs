package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"strconv"

	"gitlab.com/msiebuhr/ucs"
)

func main() {
	conn, err := net.Dial("tcp", ":8126")
	if err != nil {
		log.Fatalf("Could not connect: %s", err)
	}
	defer conn.Close()

	// Send version code and read answer
	fmt.Fprintf(conn, "%08x", 0xfe)
	versionBytes := make([]byte, 8)
	_, err = io.ReadFull(conn, versionBytes)
	if err != nil {
		log.Fatalf("Could not read returned version: %s", err)
	}
	log.Printf("Got version %s", versionBytes)

	// Make a random (failing) request
	randomGuidAndHash := make([]byte, 32)
	rand.Read(randomGuidAndHash)
	fmt.Fprintf(conn, "gi%032s", randomGuidAndHash)
	responseAndData := make([]byte, 2+32)
	_, err = io.ReadFull(conn, responseAndData)
	if err != nil {
		log.Fatalf("Could not read returned data: %s", err)
	}
	log.Printf("Got response %s %s", responseAndData[:2], ucs.PrettyUuidAndHash(responseAndData[3:]))

	// Put something in the cache
	rand.Read(randomGuidAndHash)
	fmt.Fprintf(conn, "ts%032s", randomGuidAndHash)
	data := make([]byte, 128)
	fmt.Fprintf(conn, "pi%016x%s", len(data), data)
	fmt.Fprintf(conn, "pa%016x%s", len(data), data)
	fmt.Fprintf(conn, "te")

	// Get data back
	fmt.Fprintf(conn, "gi%032s", randomGuidAndHash)
	// Initial data - hit or not
	dataType := make([]byte, 2)
	_, err = io.ReadFull(conn, dataType)
	if err != nil {
		log.Fatalf("Could not read response: %s", err)
	}
	log.Printf("Got response %s", dataType)
	if dataType[0] == '+' {
		sizeBytes := make([]byte, 16)
		_, err := io.ReadFull(conn, sizeBytes)
		if err != nil {
			log.Printf("Could not read response size: %s", err)
		}
		size, err := strconv.ParseUint(string(sizeBytes), 16, 64)
		if err != nil {
			log.Printf("Could not parse int: %s", err)
		}
		log.Printf("Got positive response size: %s/%d", sizeBytes, size)

		// Read guid + hash
		guidAndHash := make([]byte, 32)
		_, err = io.ReadFull(conn, guidAndHash)
		if err != nil {
			log.Printf("Could not read response guid+hash: %s", err)
		}
		log.Printf("Got positive response guid/hash: %s", ucs.PrettyUuidAndHash(guidAndHash))

		response := make([]byte, size)
		_, err = io.ReadFull(conn, response)
		if err != nil {
			log.Printf("Could not read data: %s", err)
		}
		// Check response data matches what we uploaded
		log.Printf("Returned data matches upload? %t!", bytes.Equal(data, response))
	} else if dataType[0] == '-' {
		// Read guid + hash
		guidAndHash := make([]byte, 32)
		_, err = io.ReadFull(conn, guidAndHash)
		if err != nil {
			log.Printf("Could not read response guid+hash: %s", err)
		}
		log.Printf("Got negative response guid/hash: %s", ucs.PrettyUuidAndHash(guidAndHash))
	}

	conn.Write([]byte("q"))
	// Read all remaining data
	rest, err := ioutil.ReadAll(conn)
	if err != nil {
		log.Fatalf("Could not read remaining data: %x", err)
	}
	log.Printf("Got rest '%s'", rest)

}
