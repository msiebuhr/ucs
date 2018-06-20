package main

import (
	"context"

	"gitlab.com/msiebuhr/ucs"
	"gitlab.com/msiebuhr/ucs/cache"

	"github.com/namsral/flag"
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

	server.Listen(context.Background(), "tcp", address)
}
