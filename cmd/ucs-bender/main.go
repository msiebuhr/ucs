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

	"github.com/namsral/flag"
	"github.com/pinterest/bender"
	"github.com/pinterest/bender/hist"
)

// Generate synthetic cache requests.
//
// TODO: IRL we generally see a "hello"-request, checking there's a cache
// server around, A new connection with tonnes of GET's for resources and then
// another connection that slowly uploads resources as the missing ones are
// generated. We should generate all of those
func SyntheticCacheRequests(n int) chan interface{} {
	c := make(chan interface{}, 100)

	guidAndHash := make([]byte, 32)
	rand.Read(guidAndHash)

	// Generate things to go on the wire
	go func() {
		for i := 0; i < n; i++ {
			buf := bytes.NewBufferString("000000fe")
			// TODO: Get/put requests

			// Generate 100 lookups
			for j := 0; j < 100; j++ {
				guidAndHash[0] = byte(i + j)
				//fmt.Fprintf(buf, "gi%s", guidAndHash);
				buf.Write([]byte("gi"))
				buf.Write(guidAndHash)
			}
			buf.Write([]byte{'q'})
			c <- buf
		}
		close(c)
	}()

	return c
}

// Executes request-series against the cache server
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

var (
	requestCount int
	workerCount  int
	verbose      bool
)

func init() {
	flag.BoolVar(&verbose, "verbose", false, "Spew more info")
	flag.IntVar(&requestCount, "requests", 100, "Total number of requests")
	flag.IntVar(&workerCount, "workers", 10, "Worker number")
}

func main() {
	flag.Parse()

	requests := SyntheticCacheRequests(requestCount)
	exec := CacheExecutor
	recorder := make(chan interface{}, requestCount)

	// Set up semaphore for parallel workers
	ws := bender.NewWorkerSemaphore()
	go func() { ws.Signal(workerCount) }()

	bender.LoadTestConcurrency(ws, requests, exec, recorder)

	l := log.New(ioutil.Discard, "", log.LstdFlags)
	if verbose {
		l = log.New(os.Stdout, "", log.LstdFlags)
	}
	h := hist.NewHistogram(60000, int(time.Millisecond))
	bender.Record(recorder, bender.NewLoggingRecorder(l), bender.NewHistogramRecorder(h))
	fmt.Println(h)

}
