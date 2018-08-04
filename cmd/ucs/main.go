package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"gitlab.com/msiebuhr/ucs"
	"gitlab.com/msiebuhr/ucs/cache"

	"github.com/namsral/flag"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	cacheBackend string
	address      string
	HTTPAddress  string
	Quota        int
	verbose      bool
)

func init() {
	flag.StringVar(&cacheBackend, "cache-backend", "fs", "Cache backend (fs or memory)")
	flag.StringVar(&address, "address", ":8126", "Address and port to listen on")
	flag.StringVar(&HTTPAddress, "http-address", ":9126", "Address and port for HTTP metrics/admin interface")
	flag.IntVar(&Quota, "quota", 1e9, "Storage quota in bytes")
	flag.BoolVar(&verbose, "verbose", false, "Spew more info")
}

func main() {
	flag.Parse()

	// Figure out a cache
	var c cache.Cacher
	switch cacheBackend {
	case "fs":
		var err error
		c, err = cache.NewFS()
		if err != nil {
			panic(err)
		}
	case "memory":
		c = cache.NewMemory(Quota)
	default:
		// UNKNOWN BACKEND - BAIL/CRASH/QUIT
		panic("Unknown backend " + cacheBackend)
	}

	server := ucs.NewServer(
		func(s *ucs.Server) { s.Cache = c },
		func(s *ucs.Server) {
			if verbose {
				s.Log = log.New(os.Stdout, "server: ", 0)
			}
		},
	)

	// Expose metrics through an HTTP server
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		if err := http.ListenAndServe(HTTPAddress, nil); err != nil {
			log.Println("ListenAndServe: ", err)
		}
	}()

	server.Listen(context.Background(), "tcp", address)
}
