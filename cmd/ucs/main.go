package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/msiebuhr/ucs"
	"github.com/msiebuhr/ucs/cache"
	"github.com/msiebuhr/ucs/customflags"
	"github.com/msiebuhr/ucs/frontend"

	"github.com/namsral/flag"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	cacheBackend string
	HTTPAddress  string
	quota        = customflags.NewSize(1024 * 1024 * 1024)
	verbose      bool
	ports        = &customflags.Namespaces{}
)

func init() {
	flag.StringVar(&cacheBackend, "cache-backend", "fs", "Cache backend (fs or memory)")
	flag.StringVar(&HTTPAddress, "http-address", ":9126", "Address and port for HTTP metrics/admin interface")
	flag.BoolVar(&verbose, "verbose", false, "Spew more info")
	flag.Var(quota, "quota", "Storage quota (ex. 10GB, 1TB, ...)")
	flag.Var(ports, "port", "Namespaces/ports to open (ex: zombie-zebras:5000) May be used multiple times")
}

func main() {
	flag.Parse()

	// Set a defalt port if the user doesn't set anything
	if len(*ports) == 0 {
		ports.Set("default:8126")
	}

	log.Println("Starting. Quota ", quota)
	log.Println("Starting. Ports ", ports)

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

	// Create a server per namespace
	servers := make([]*ucs.Server, 0, len(*ports))
	for ns, port := range *ports {
		server := ucs.NewServer(
			func(s *ucs.Server) { s.Cache = c },
			func(s *ucs.Server) {
				if verbose {
					s.Log = log.New(os.Stdout, "server: ", 0)
				}
			},
			func(s *ucs.Server) { s.Namespace = ns },
		)
		servers = append(servers, server)
		go server.Listen(context.Background(), fmt.Sprintf(":%d", port))
	}

	// Set up web-server mux
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/", http.FileServer(frontend.FS(false)))

	// Create the web-server itself
	h := &http.Server{Addr: HTTPAddress, Handler: mux}

	// Start it
	go func() {
		if err := h.ListenAndServe(); err != nil {
			log.Println("ListenAndServe: ", err)
		}
	}()

	// Handle SIGINT and SIGTERM.
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	log.Println(<-ch)

	// Stop web interface gracefully
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	h.Shutdown(ctx)

	// Stop the service gracefully.
	for _, server := range servers {
		server.Stop()
	}
}
