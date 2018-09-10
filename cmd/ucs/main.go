package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"gitlab.com/msiebuhr/ucs"
	"gitlab.com/msiebuhr/ucs/cache"

	"github.com/docker/go-units"
	"github.com/namsral/flag"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Quick-and-dirty human-readable sizes
type Size struct {
	size *int64
}

func (v Size) String() string {
	if v.size == nil {
		return ""
	}
	return units.HumanSize(float64(*v.size))
}

func (v Size) Int64() int64 {
	if v.size == nil {
		return 0
	}
	return *v.size
}

func (v Size) Set(s string) error {
	b, err := units.FromHumanSize(s)
	if err != nil {
		return err
	}
	*v.size = b
	return nil
}

func NewSize(s int64) *Size {
	size := Size{}
	size.size = &s
	return &size
}

var (
	cacheBackend string
	address      string
	HTTPAddress  string
	quota        = NewSize(1e9)
	verbose      bool
)

func init() {
	flag.StringVar(&cacheBackend, "cache-backend", "fs", "Cache backend (fs or memory)")
	flag.StringVar(&address, "address", ":8126", "Address and port to listen on")
	flag.StringVar(&HTTPAddress, "http-address", ":9126", "Address and port for HTTP metrics/admin interface")
	flag.BoolVar(&verbose, "verbose", false, "Spew more info")
	flag.Var(quota, "quota", "Storage quota (ex. 10GB, 1TB, ...)")
}

func main() {
	flag.Parse()

	log.Println("Starting. Quota ", quota)

	// Figure out a cache
	var c cache.Cacher
	switch cacheBackend {
	case "fs":
		var err error
		c, err = cache.NewFS(func(f *cache.FS) { f.Quota = quota.Int64() })
		if err != nil {
			panic(err)
		}
	case "memory":
		c = cache.NewMemory(quota.Int64())
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
