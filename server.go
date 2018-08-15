package ucs

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"strconv"
	"time"

	"gitlab.com/msiebuhr/ucs/cache"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	ops = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "ucs_server_ops",
		Help: "Operations performed on the server",
	}, []string{"op"})
	getCacheHit = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "ucs_server_get_hits",
		Help: "Hit/miss upon get'ing from the cache",
	}, []string{"type", "hit"})
	getBytes = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: "ucs_server_get_bytes",
		Help: "Bytes fetched fom server",
	}, []string{"type"})
	putBytes = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: "ucs_server_put_bytes",
		Help: "Bytes sent fom server",
	}, []string{"type"})
	getDurations = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: "ucs_server_get_duration_seconds",
		Help: "Time spent sending data",
	}, []string{"type"})
	putDurations = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: "ucs_server_put_duration_seconds",
		Help: "Time spent recieving data",
	}, []string{"type"})
)

func init() {
	prometheus.MustRegister(ops)
	prometheus.MustRegister(getCacheHit)
	prometheus.MustRegister(getBytes)
	prometheus.MustRegister(putBytes)
	prometheus.MustRegister(getDurations)
	prometheus.MustRegister(putDurations)
}

func PrettyUuidAndHash(d []byte) string {
	return fmt.Sprintf("%x/%x", d[:16], d[17:])
}

type Server struct {
	Cache cache.Cacher
	Log   *log.Logger
}

// Set up a new server
func NewServer(options ...func(*Server)) *Server {
	s := &Server{
		Cache: cache.NewNOP(),
		Log:   log.New(ioutil.Discard, "", 0),
	}

	for _, f := range options {
		f(s)
	}

	return s
}

func (s *Server) Listen(ctx context.Context, network, address string) error {
	listener, err := net.Listen(network, address)
	if err != nil {
		s.log(ctx, "Error listening:", err.Error())
		return err
	}
	defer listener.Close()
	s.logf(ctx, "Listening on %s", listener.Addr())

	return s.Listener(ctx, listener)
}

func (s *Server) Listener(ctx context.Context, listener net.Listener) error {
	for {
		// Listen for an incoming connection.
		conn, err := listener.Accept()
		if err != nil {
			s.log(ctx, "Error accepting: ", err.Error())
			continue
		}
		// Handle connections in a new goroutine.
		connCtx := context.WithValue(ctx, "addr", conn.RemoteAddr().String())
		go s.handleRequest(connCtx, conn)
	}
}

func (s *Server) log(ctx context.Context, rest ...interface{}) {
	// Extract and sort values from ctx
	values := make([]interface{}, 0)

	keys := []string{"namespace", "addr"}
	for _, key := range keys {
		value := ctx.Value(key)
		if (value!=nil) {
			values = append(values, fmt.Sprintf("%s=%s", key, value))
		}
	}

	values = append(values, rest...)

	s.Log.Println(values...)
}

func (s *Server) logf(ctx context.Context, format string, rest ...interface{}) {
	s.log(ctx, fmt.Sprintf(format, rest...))
}

func readVersionNumber(rw *bufio.ReadWriter) (uint32, error) {
	// Peek at some data so we force it to wait for some data so we don't get
	// zero buffer sizes all the time...
	rw.Reader.Peek(2)

	// See how much data is waiting for us.
	bytesToRead := rw.Reader.Buffered()
	// We want to read at most eight hex decimals
	if bytesToRead > 8 {
		bytesToRead = 8
	}
	// And at least two.
	if bytesToRead < 2 {
		bytesToRead = 2
	}

	bytes := make([]byte, bytesToRead)
	_, err := io.ReadFull(rw, bytes)
	if err != nil {
		return 0, err
	}

	// Convert number from hex to a real one
	n, err := strconv.ParseUint(string(bytes), 16, 32)
	return uint32(n), err
}

// Handles incoming requests.
func (s *Server) handleRequest(ctx context.Context, conn net.Conn) {
	start := time.Now()
	defer func() {
		s.log(ctx, "Closing connection")
		conn.Close()
	}()
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	defer rw.Flush()

	// Set deadline for getting data five seconds in the future
	conn.SetDeadline(time.Now().Add(30 * time.Second))

	trx := make([]byte, 0)
	trxData := cache.Line{}

	// First, read uint32 version number
	version, err := readVersionNumber(rw)
	if err != nil {
		s.logf(ctx, "Could not read client version: %s", err)
		return
	}
	ctx = context.WithValue(ctx, "version", version)

	// Bail on unknown versions
	if version != 0xfe {
		s.logf(ctx, "Got invalid client version %d", version)
		fmt.Fprintf(rw, "%08x", 0)
		return
	}

	// Protocol says to echo version if everything is ok
	s.logf(ctx, "Got client version %d", version)
	fmt.Fprintf(rw, "%08x", version)

	for {
		// Extend Read-dealine by five seconds for each command we process
		conn.SetDeadline(time.Now().Add(30 * time.Second))

		// Flush version or previous command
		// The original server get really confused if clients do agressive
		// streaming. Explicitly waiting for output from previous command
		// seem to make it happy...
		rw.Flush()

		cmd, err := rw.ReadByte()
		if err == io.EOF {
			s.log(ctx, "Client hangup; Quitting")
			return
		} else if err != nil {
			s.log(ctx, "Error reading command:", err)
			return
		}

		// Quit command
		if cmd == 'q' {
			ops.WithLabelValues("q").Inc()
			s.logf(ctx, "Got command '%c'; Quitting", cmd)
			return
		}

		// Get type
		cmdType, err := rw.ReadByte()
		if err != nil {
			s.log(ctx, "Error reading command type:", err)
			return
		}
		s.logf(ctx, "Got command '%c'/'%c'", cmd, cmdType)

		start = time.Now()

		// GET
		if cmd == 'g' {
			ops.WithLabelValues("g").Inc()

			// Read uuidAndHash
			uuidAndHash := make([]byte, 32)
			_, err := io.ReadFull(rw, uuidAndHash)

			s.logf(ctx, "Get '%c' '%s'", cmdType, PrettyUuidAndHash(uuidAndHash))

			data, err := s.Cache.Get(cache.Kind(cmdType), uuidAndHash)
			if err != nil {
				getCacheHit.WithLabelValues(string(cmdType), "miss").Inc()
				s.log(ctx, "Error reading from cache:", err)
				fmt.Fprintf(rw, "-%c%s", cmdType, uuidAndHash)
				continue
			}
			if len(data) == 0 {
				getCacheHit.WithLabelValues(string(cmdType), "miss").Inc()
				s.log(ctx, "Cache miss")
				fmt.Fprintf(rw, "-%c%s", cmdType, uuidAndHash)
				continue
			}

			fmt.Fprintf(rw, "+%c%016x%s", cmdType, len(data), uuidAndHash)
			rw.Write(data)
			getCacheHit.WithLabelValues(string(cmdType), "hit").Inc()
			getBytes.WithLabelValues(string(cmdType)).Observe(float64(len(data)))
			getDurations.WithLabelValues(string(cmdType)).Observe(time.Now().Sub(start).Seconds())
			continue
		}

		// Transaction start
		if cmd == 't' && cmdType == 's' {
			ops.WithLabelValues("ts").Inc()

			// Bail if we're already in a command
			if len(trx) > 0 {
				s.log(ctx, "Error starting trx inside trx")
				return
			}

			// Read uuidAndHash
			uuidAndHash := make([]byte, 32)
			_, err := io.ReadFull(rw, uuidAndHash)
			if err != nil {
				s.log(ctx, "Error reading uuid+hash:", err)
				return
			}

			s.logf(ctx, "Transaction started for %s", PrettyUuidAndHash(uuidAndHash))

			trx = uuidAndHash
			continue
		}

		// Transaction end
		if cmd == 't' && cmdType == 'e' {
			ops.WithLabelValues("te").Inc()

			if len(trx) == 0 {
				s.log(ctx, "Error ending trx - none started")
				return
			}

			err := s.Cache.Put(trx, trxData)
			if err != nil {
				s.log(ctx, "Error ending trx - cache put error:", err)
				continue
			}

			trx = []byte{}
			trxData = cache.Line{}
			continue
		}

		// Put
		if cmd == 'p' {
			ops.WithLabelValues("p").Inc()

			// Bail on wrong CMD-types
			if cmdType != 'a' && cmdType != 'i' && cmdType != 'r' {
				s.logf(ctx, "Error putting - invalid type %s", []byte{cmdType})
				return
			}

			// Read size
			sizeBytes := make([]byte, 16)
			_, err := io.ReadFull(rw, sizeBytes)
			if err != nil {
				s.log(ctx, "Error putting - cannot read size:", err)
				return
			}

			// Parse size
			size, err := strconv.ParseUint(string(sizeBytes), 16, 64)
			if err != nil {
				s.logf(ctx, "Error putting - cannot parse size '%x': %s", sizeBytes, err)
				return
			}
			s.log(ctx, "Put, size", string(sizeBytes), size)

			// TODO: Cache should probably have the reader embedded
			trxData.PutReader(cache.Kind(cmdType), size, rw)

			putBytes.WithLabelValues(string(cmdType)).Observe(float64(size))
			putDurations.WithLabelValues(string(cmdType)).Observe(time.Now().Sub(start).Seconds())
			continue
		}

		// Invalid command
		ops.WithLabelValues("invalid").Inc()
		s.logf(ctx, "Invalid command: %c%c", cmd, cmdType)
		return
	}
}
