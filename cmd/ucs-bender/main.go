package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/msiebuhr/ucs"
	"github.com/msiebuhr/ucs/cache"

	"github.com/namsral/flag"
	"github.com/pinterest/bender"
	"github.com/pinterest/bender/hist"
)

// Generate synthetic cache requests.
//
// Traces from Unity shows connections generally come in triplets; One at
// startup, which immediately quits (presumably to check cache server
// availability). Then when Unity has figured out what assets it needs, another
// connection is opened up, which requests all the assets. Finally, if any
// assets were missing, a connection to upload these after they've been built locally.
func SyntheticCacheRequests(n int) chan interface{} {
	c := make(chan interface{}, 100)

	guidAndHash := make([]byte, 32)
	rand.Read(guidAndHash)

	go func() {
		for i := 0; i < n; i++ {
			reqs := []ucs.CacheRequester{}

			switch i % 3 {
			case 0:
				// No requests -- just a cache-aliveness request
			case 1:
				// Tonnes of GET-requests
				for j := 0; j < 10; j++ {
					guidAndHash[0] = byte(i + j)
					reqs = append(
						reqs,
						ucs.Get(cache.KIND_INFO, guidAndHash),
						ucs.Get(cache.KIND_ASSET, guidAndHash),
					)
				}
			case 2:
				// Put-requests
				for j := 0; j < 10; j++ {
					guidAndHash[0] = byte(i + j)
					reqs = append(
						reqs,
						ucs.Put(guidAndHash, ucs.PutString("info"), ucs.PutString("asset"), nil),
					)
				}
			}
			c <- reqs
		}
		close(c)
	}()

	return c
}

// Executes request-series against the cache server
func CacheExecutor(unix_nsec int64, transport interface{}) (interface{}, error) {
	// Convert transport into cache requests
	buf, ok := transport.([]ucs.CacheRequester)
	if !ok {
		return nil, errors.New("Transport was not []CacheRequester")
	}

	// Create buffered connection
	conn, err := net.Dial("tcp", ":8126")
	if err != nil {
		return nil, err
		//log.Fatalf("Could not connect: %s", err)
	}

	c := ucs.NewClient(conn)
	_, err = c.NegotiateVersion(254)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	// Execute requests
	for _, req := range buf {
		// Check if it is a PUT or GET request
		if get, ok := req.(*ucs.GetRequest); ok {
			go io.Copy(ioutil.Discard, get)
			/*
				go func () {
					data, err := ioutil.ReadAll(get)
					if err != nil { fmt.Printf("err: %s", err) }
					fmt.Printf("  data: %db", len(data))
				}()
			*/
			err = c.Execute(get)
		} else {
			err = c.Execute(req)
		}
		if (err != nil) {
			return nil, err
		}
	}

	// Return output
	return nil, nil
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
