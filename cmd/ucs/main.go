package main

import (
	"context"
	"log"
	"net/http"

	"gitlab.com/msiebuhr/ucs"
	"gitlab.com/msiebuhr/ucs/cache"

	"github.com/namsral/flag"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	cacheBackend string
	address      string
)

func init() {
	flag.StringVar(&cacheBackend, "cache-backend", "fs", "Cache backend (fs or memory)")
	flag.StringVar(&address, "address", ":8126", "Address and port to listen on")
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
		c = cache.NewMemory()
	default:
		// UNKNOWN BACKEND - BAIL/CRASH/QUIT
		panic("Unknown backend " + cacheBackend)
	}

	server := ucs.NewServer(
		func(s *ucs.Server) { s.Cache = c },
	)

	// Expose metrics through an HTTP server
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Println("ListenAndServe: ", err)
		}
	}()

	server.Listen(context.Background(), "tcp", address)
}
