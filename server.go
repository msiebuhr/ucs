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
	"sync"
	"time"

	"github.com/msiebuhr/ucs/cache"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	ops = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "ucs_server_ops",
		Help: "Operations performed on the server",
	}, []string{"namespace", "op"})
	// Technically not needed, as `getBytes_sum / ops{op="g"}` would produce ca. same results
	getCacheHit = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "ucs_server_get_hits",
		Help: "Hit/miss upon get'ing from the cache",
	}, []string{"namespace", "type", "hit"})
	getBytes = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: "ucs_server_get_bytes",
		Help: "Bytes fetched fom server",
	}, []string{"namespace", "type"})
	putBytes = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: "ucs_server_put_bytes",
		Help: "Bytes sent fom server",
	}, []string{"namespace", "type"})
	getDurations = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: "ucs_server_get_duration_seconds",
		Help: "Time spent sending data",
	}, []string{"namespace", "type"})
	putDurations = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: "ucs_server_put_duration_seconds",
		Help: "Time spent receiving data",
	}, []string{"namespace", "type"})
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
	Cache     cache.Cacher
	Log       *log.Logger
	Namespace string

	closer    chan bool
	waitGroup *sync.WaitGroup
}

// Set up a new server
func NewServer(options ...func(*Server)) *Server {
	s := &Server{
		Cache:     cache.NewNOP(),
		Log:       log.New(ioutil.Discard, "", 0),
		Namespace: "",
		closer:    make(chan bool, 1),
		waitGroup: &sync.WaitGroup{},
	}

	for _, f := range options {
		f(s)
	}

	return s
}

func (s *Server) Listen(ctx context.Context, address string) error {
	laddr, err := net.ResolveTCPAddr("tcp", address)
	if nil != err {
		s.log(ctx, "Error resolving address:", err.Error())
		return err
	}
	listener, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		s.log(ctx, "Error listening:", err.Error())
		return err
	}
	defer listener.Close()
	s.logf(ctx, "Listening on %s", listener.Addr())

	return s.Listener(ctx, listener)
}

func (s *Server) Listener(ctx context.Context, listener *net.TCPListener) error {
	// Enrich context with current namespace
	if s.Namespace != "" {
		ctx = context.WithValue(ctx, "namespace", s.Namespace)
	}

	// Initialize metrics with given labels
	for _, op := range []string{"g", "p", "ts", "te"} {
		ops.WithLabelValues(s.Namespace, op)
	}
	for _, kind := range []string{string(cache.KIND_ASSET), string(cache.KIND_INFO), string(cache.KIND_RESOURCE)} {
		getCacheHit.WithLabelValues(s.Namespace, (kind), "hit")
		getCacheHit.WithLabelValues(s.Namespace, (kind), "miss")
		getBytes.WithLabelValues(s.Namespace, (kind))
		getDurations.WithLabelValues(s.Namespace, (kind))
		putBytes.WithLabelValues(s.Namespace, (kind))
		putDurations.WithLabelValues(s.Namespace, (kind))
	}

	for {
		select {
		case <-s.closer:
			s.log(ctx, "Stopping listening")
			listener.Close()
			return nil
		default:
		}
		listener.SetDeadline(time.Now().Add(1 * time.Second))
		conn, err := listener.AcceptTCP()
		if nil != err {
			// Ignore timeout errors -- we have those so we'll also take a peek at the closer-channel.
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				continue
			}

			// Genuine error - log and try again
			s.log(ctx, "Error accepting: ", err.Error())
			continue
		}
		//s.waitGroup.Add(1)
		connCtx := context.WithValue(ctx, "addr", conn.RemoteAddr().String())
		s.log(connCtx, "Connected")
		go s.handleRequest(connCtx, conn)
	}
}

func (s *Server) Stop() {
	close(s.closer)
	s.waitGroup.Wait()
}

func (s *Server) log(ctx context.Context, rest ...interface{}) {
	// Extract and sort values from ctx
	values := make([]interface{}, 0)

	keys := []string{"namespace", "addr"}
	for _, key := range keys {
		value := ctx.Value(key)
		if value != nil {
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

type serverGetRequest struct {
	kind        cache.Kind
	uuidAndHash []byte
}

// Responds to Get requests queued up in the reqs-channel.
func (s *Server) respondToGetRequests(ctx context.Context, w io.Writer, reqs chan *serverGetRequest) error {
	var start time.Time

	for req := range reqs {
		start = time.Now()
		size, reader, err := s.Cache.Get(s.Namespace, req.kind, req.uuidAndHash)
		/*
			s.logf(
				ctx,
				"Get kind=%c uuidAndHash=%s size=%d err=%v hit=%t",
				req.kind, PrettyUuidAndHash(req.uuidAndHash), size, err, size > 0 && err == nil,
			)
		*/

		// Treat internal errors as MISS
		if err != nil {
			s.log(ctx, "Error getting from cache:", err)
		}

		if err != nil || size == 0 {
			getCacheHit.WithLabelValues(s.Namespace, string(req.kind), "miss").Inc()
			_, err = fmt.Fprintf(w, "-%c%s", req.kind, req.uuidAndHash)
			if err != nil {
				return err
			}
			if reader != nil {
				reader.Close()
			}
			continue
		}

		// Everything's A-OK
		_, err = fmt.Fprintf(w, "+%c%016x%s", req.kind, size, req.uuidAndHash)
		if err != nil {
			return err
		}
		_, err = io.Copy(w, reader)
		if err != nil {
			return err
		}
		reader.Close()
		getCacheHit.WithLabelValues(s.Namespace, string(req.kind), "hit").Inc()
		getBytes.WithLabelValues(s.Namespace, string(req.kind)).Observe(float64(size))
		getDurations.WithLabelValues(s.Namespace, string(req.kind)).Observe(time.Now().Sub(start).Seconds())
	}
	return nil
}

// Handles incoming requests.
func (s *Server) handleRequest(ctx context.Context, conn net.Conn) {
	s.waitGroup.Add(1)
	start := time.Now()
	readerAndWriterDone := sync.WaitGroup{}
	getRequests := make(chan *serverGetRequest, 100000)

	defer func() {
		s.log(ctx, "Closing connection")
		close(getRequests)
		readerAndWriterDone.Wait()
		conn.Close()
		s.waitGroup.Done()
	}()
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	defer rw.Flush()

	// Set 30s deadline for getting handshake done
	conn.SetDeadline(time.Now().Add(30 * time.Second))

	var trx cache.Transaction
	defer func() {
		if trx != nil {
			trx.Abort()
		}
	}()

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

	// Flush version, so client will begin sending data
	rw.Flush()

	// Now that we've done a handshake (and sent it), we begin doing async
	// reading/writing, as the Unity editor wants to send *all* its
	// get-requests before listening for responses.
	readerAndWriterDone.Add(1)
	go func(reqs chan *serverGetRequest) {
		defer readerAndWriterDone.Done()

		sendError := s.respondToGetRequests(ctx, conn, reqs)
		s.logf(ctx, "Done sending data err=%s", sendError)
	}(getRequests)

	for {
		// Extend Read-dealine by five minutes for each command we process
		conn.SetDeadline(time.Now().Add(5 * time.Minute))

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
			ops.WithLabelValues(s.Namespace, "q").Inc()
			s.logf(ctx, "Got command '%c'; Quitting", cmd)
			return
		}

		// Get type
		cmdType, err := rw.ReadByte()
		if err != nil {
			s.log(ctx, "Error reading command type:", err)
			return
		}
		s.logf(ctx, "Got command op=%c kind=%c", cmd, cmdType)

		start = time.Now()

		// GET
		if cmd == 'g' {
			ops.WithLabelValues(s.Namespace, "g").Inc()

			// Read uuidAndHash
			uuidAndHash := make([]byte, 32)
			_, err := io.ReadFull(rw, uuidAndHash)
			if err != nil {
				s.logf(ctx, "Error reading: %s", err)
			}

			//s.logf(ctx, "Get request parsed kind=%c uuidAndHash=%s", cmdType, PrettyUuidAndHash(uuidAndHash))
			getRequests <- &serverGetRequest{
				kind:        cache.Kind(cmdType),
				uuidAndHash: uuidAndHash,
			}

			continue
		}

		// Transaction start
		if cmd == 't' && cmdType == 's' {
			ops.WithLabelValues(s.Namespace, "ts").Inc()

			// Bail if we're already in a command
			if trx != nil {
				s.log(ctx, "Transaction start error: Already in transaction")
				return
			}

			// Read uuidAndHash
			uuidAndHash := make([]byte, 32)
			_, err := io.ReadFull(rw, uuidAndHash)
			if err != nil {
				s.log(ctx, "Error reading uuid+hash:", err)
				return
			}

			//s.logf(ctx, "Transaction start uuidAndHash=%s", PrettyUuidAndHash(uuidAndHash))

			trx = s.Cache.PutTransaction(s.Namespace, uuidAndHash)
			continue
		}

		// Transaction end
		if cmd == 't' && cmdType == 'e' {
			ops.WithLabelValues(s.Namespace, "te").Inc()

			if trx == nil {
				s.log(ctx, "Transaction end error: None started")
				return
			}

			err := trx.Commit()
			if err != nil {
				s.log(ctx, "Transaction end error: Commit failed:", err)
				continue
			}

			s.log(ctx, "Transaction end")

			trx = nil
			continue
		}

		// Put
		if cmd == 'p' {
			ops.WithLabelValues(s.Namespace, "p").Inc()

			// Bail on wrong CMD-types
			if cmdType != 'a' && cmdType != 'i' && cmdType != 'r' {
				s.logf(ctx, "Put error: invalid type '%s'", []byte{cmdType})
				return
			}

			if trx == nil {
				s.logf(ctx, "Put error: Not inside transaction")
				return
			}

			// Read size
			sizeBytes := make([]byte, 16)
			_, err := io.ReadFull(rw, sizeBytes)
			if err != nil {
				s.log(ctx, "Put error: cannot read size:", err)
				return
			}

			// Parse size
			size, err := strconv.ParseUint(string(sizeBytes), 16, 64)
			if err != nil {
				s.logf(ctx, "Put error: cannot parse size '%x': %s", sizeBytes, err)
				return
			}
			s.logf(ctx, "Put kind=%c size=%d", cmdType, size)

			trx.Put(int64(size), cache.Kind(cmdType), io.LimitReader(rw, int64(size)))

			putBytes.WithLabelValues(s.Namespace, string(cmdType)).Observe(float64(size))
			putDurations.WithLabelValues(s.Namespace, string(cmdType)).Observe(time.Now().Sub(start).Seconds())
			continue
		}

		// Invalid command
		ops.WithLabelValues(s.Namespace, "invalid").Inc()
		s.logf(ctx, "Invalid command: %c%c", cmd, cmdType)
		return
	}
}
