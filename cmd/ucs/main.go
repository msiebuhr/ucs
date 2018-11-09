package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
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
	cacheBackend    string
	fsCacheBasepath string
	HTTPAddress     string
	quota           = customflags.NewSize(1024 * 1024 * 1024)
	verbose         bool
	ports           = &customflags.Namespaces{}
)

func init() {
	flag.StringVar(&cacheBackend, "cache-backend", "fs", "Cache backend (fs or memory)")
	flag.StringVar(&fsCacheBasepath, "cache-path", "./unity-cache", "Where FS cache should store data")
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

	log.Printf(
		"Starting quota=%s ports=%s httpAddress=%s fsCacheBasepath=%s\n",
		quota, ports, HTTPAddress, fsCacheBasepath,
	)

	// Figure out a cache
	var c cache.Cacher
	switch cacheBackend {
	case "fs":
		var err error
		c, err = cache.NewFS(func(f *cache.FS) {
			f.Quota = quota.Int64()
			f.Basepath = fsCacheBasepath
		})
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
		go func(port uint) {
			err := server.Listen(context.Background(), fmt.Sprintf(":%d", port))
			log.Fatalln("Listen:", err)
		}(port)
	}

	// Set up web-server mux
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/", http.FileServer(frontend.FS(false)))
	mux.HandleFunc("/api/info", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		// Figure out our IP
		var ip net.IP
		addrs, _ := net.InterfaceAddrs()
		for _, addr := range addrs {
			var i net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				i = v.IP
			case *net.IPAddr:
				i = v.IP
			}
			// Is this the right way to detect the IP?
			if i.IsGlobalUnicast() {
				ip = i
				break
			}
		}

		servers := map[string]string{}
		for ns, port := range *ports {
			// Parse address to figure out what port/ip we're bound to
			tcpAddr := net.TCPAddr{
				IP:   ip,
				Port: int(port),
			}
			servers[ns] = tcpAddr.String()
		}

		data := struct {
			QuotaBytes   int64
			Servers      map[string]string
			CacheBackend string
		}{
			QuotaBytes:   quota.Int64(),
			Servers:      servers,
			CacheBackend: cacheBackend,
		}

		e := json.NewEncoder(w)
		e.Encode(data)
	})

	// Create the web-server itself
	h := &http.Server{Addr: HTTPAddress, Handler: mux}

	// Start it
	go func() {
		if err := h.ListenAndServe(); err != nil {
			log.Fatalln("ListenAndServe: ", err)
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
