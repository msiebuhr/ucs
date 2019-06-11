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
	"github.com/msiebuhr/ucs/customflags"

	"github.com/docker/go-units"
	"github.com/namsral/flag"
	"github.com/pinterest/bender"
	"github.com/pinterest/bender/hist"
)

func PutRandom(s int64) *ucs.PutObject {
	reader := io.LimitReader(
		rand.New(rand.NewSource(s)),
		s,
	)

	return ucs.NewPutObject(
		reader,
		int(s),
	)
}

// Generate synthetic cache requests.
//
// Traces from Unity shows connections generally come in triplets; One at
// startup, which immediately quits (presumably to check cache server
// availability). Then when Unity has figured out what assets it needs, another
// connection is opened up, which requests all the assets. Finally, if any
// assets were missing, a connection to upload these after they've been built locally.
func SyntheticCacheRequests(n int, commands int, size int64) chan interface{} {
	c := make(chan interface{}, 100)

	guidAndHash := make([]byte, 32)
	rand.Read(guidAndHash)

	go func() {
		for i := 0; i < n; i++ {
			halfPipe, _ := net.Pipe()
			client := ucs.NewBulkClientConn(halfPipe)

			switch i % 3 {
			case 0:
				// No requests -- just a cache-aliveness request
			case 1:
				// Tonnes of GET-requests
				for j := 0; j < commands; j++ {
					guidAndHash[0] = byte(i + j)
					client.Get(cache.KIND_INFO, guidAndHash)
					client.Get(cache.KIND_ASSET, guidAndHash)
				}
			case 2:
				// Put-requests
				for j := 0; j < commands; j++ {
					guidAndHash[0] = byte(i + j)
					client.Put(
						guidAndHash,
						ucs.PutString("info"),
						PutRandom(size), //Asset
						nil,
					)
				}
			}
			c <- client
		}
		close(c)
	}()

	return c
}

// Executes request-series against the cache server
func CacheExecutor(unix_nsec int64, transport interface{}) (interface{}, error) {
	// Convert transport into cache requests
	c, ok := transport.(*ucs.BulkClient)
	if !ok {
		return nil, errors.New("Transport was not *ucs.BulkClient")
	}

	// Create buffered connection
	conn, err := net.Dial("tcp", ":8126")
	if err != nil {
		return nil, err
		//log.Fatalf("Could not connect: %s", err)
	}
	defer conn.Close()

	c.Conn = conn
	c.Callback = func(k cache.Kind, uuidAndHash []byte, hit bool, data io.Reader) {
		io.Copy(ioutil.Discard, data)
	}

	_, err = c.NegotiateVersion(254)
	if err != nil {
		return nil, err
	}
	//defer c.Close()

	// Execute requests
	err = c.Execute()
	if err != nil {
		return nil, err
	}

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
	commandCount int
	size         = customflags.NewSize(1024 * 1024)
)

func init() {
	flag.BoolVar(&verbose, "verbose", false, "Spew more info")
	flag.IntVar(&requestCount, "requests", 100, "Total number of requests")
	flag.IntVar(&workerCount, "workers", 10, "Worker number")
	flag.IntVar(&commandCount, "commands", 100, "Commands executed per request")
	flag.Var(size, "size", "Request blob size")
}

func main() {
	flag.Parse()

	log.Println("Starting")
	log.Println(
		"Starting",
		"Requests=", requestCount,
		"Workers=", workerCount,
		"Commands/request=", commandCount,
		"AssetSize=", size,
	)
	log.Println(
		"Est. total upload=", units.BytesSize(float64((requestCount/3)*commandCount*int(size.Int64()))),
	)

	requests := SyntheticCacheRequests(requestCount, commandCount, size.Int64())
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
