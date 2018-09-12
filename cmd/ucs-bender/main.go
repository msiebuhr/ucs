package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/pinterest/bender"
	"github.com/pinterest/bender/hist"
)

// Make every n requests (configurable percentage as PUT's?)
func SyntheticCacheRequests(n int) chan interface{} {
	c := make(chan interface{}, 100)

	guidAndHash := make([]byte, 32)
	rand.Read(guidAndHash)

	// Generate things to go on the wire
	go func() {
		for i := 0; i < n; i++ {
			buf := bytes.NewBufferString("000000feq")
			// TODO: Get/put requests

			// Generate 100 lookups
			for j := 0; j < 100; j++ {
				guidAndHash[0] = byte(i + j)
				//fmt.Fprintf(buf, "gi%s", guidAndHash);
				buf.Write([]byte("gi"))
				buf.Write(guidAndHash)
			}
			c <- buf
		}
		close(c)
	}()

	return c
}

// TODO: This should be generated, so we can inject our validator
func CacheExecutor(unix_nsec int64, transport interface{}) (interface{}, error) {
	// Convert transport into bytes.Buffer
	buf, ok := transport.(*bytes.Buffer)
	if !ok {
		return nil, errors.New("Transport was not a bytes.Buffer")
	}

	// Create buffered connection
	conn, err := net.Dial("tcp", ":8126")
	if err != nil {
		log.Fatalf("Could not connect: %s", err)
	}
	defer conn.Close()

	// Execute requests
	go buf.WriteTo(conn)

	// Return output
	return ioutil.ReadAll(conn)
}

/*
func CacheValidator(request interface{}) error {
	log.Println("Validate", request)
	return nil
}
*/

func main() {
	intervals := bender.ExponentialIntervalGenerator(10)
	requests := SyntheticCacheRequests(100)
	exec := CacheExecutor
	recorder := make(chan interface{}, 100)

	bender.LoadTestThroughput(intervals, requests, exec, recorder)

	l := log.New(os.Stdout, "", log.LstdFlags)
	//l := log.New(ioutil.Discard, "", log.LstdFlags)
	h := hist.NewHistogram(60000, int(time.Millisecond))
	bender.Record(recorder, bender.NewLoggingRecorder(l), bender.NewHistogramRecorder(h))
	fmt.Println(h)

}
