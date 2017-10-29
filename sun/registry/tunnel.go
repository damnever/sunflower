package registry

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/hashicorp/yamux"
	"go.uber.org/zap"

	"github.com/damnever/sunflower/log"
	connutil "github.com/damnever/sunflower/pkg/conn"
	"github.com/damnever/sunflower/sun/tracker"
)

// TODO(damnever): close then clean up

type Tunnel interface {
	NewSession(conn net.Conn) bool
	Serve() error
	Close()
	WaitAndCleanup()
}

type TCPTunnel struct {
	*tcpBasedTunnel
}

func NewTCPTunnel(tracker *tracker.TunnelTracker, serverAddr string) (*TCPTunnel, error) {
	l, err := net.Listen("tcp", serverAddr)
	if err != nil {
		tracker.OnError(fmt.Sprintf("listen on server address: %v", err))
		return nil, err
	}
	return &TCPTunnel{
		tcpBasedTunnel: newTCPBasedTunnel(tracker, l),
	}, nil
}

type HTTPTunnel struct {
	*tcpBasedTunnel
}

func NewHTTPTunnel(tracker *tracker.TunnelTracker, l net.Listener) *HTTPTunnel {
	return &HTTPTunnel{
		tcpBasedTunnel: newTCPBasedTunnel(tracker, l),
	}
}

type tcpBasedTunnel struct {
	sync.RWMutex
	sync.WaitGroup

	logger  *zap.SugaredLogger
	server  net.Listener
	session *yamux.Session
	sesVer  uint64 // Make session versionable, since OpenStream() may block the entire lock process
	closed  bool
	tracker *tracker.TunnelTracker
}

func newTCPBasedTunnel(tracker *tracker.TunnelTracker, l net.Listener) *tcpBasedTunnel {
	tracker.Opened()
	return &tcpBasedTunnel{
		tracker: tracker,
		logger:  log.New("tnl[%s/%s]", tracker.AgentHash(), tracker.Hash()),
		server:  l,
		session: nil,
		sesVer:  0,
		closed:  false,
	}
}

func (tt *tcpBasedTunnel) NewSession(conn net.Conn) bool {
	tt.Lock()
	defer tt.Unlock()
	if tt.closed {
		tt.logger.Debugf("Tunnel closed")
		return false
	}
	if tt.session != nil {
		if !tt.session.IsClosed() {
			return false
		}
		tt.logger.Debugf("Session already closed")
	}
	conn.SetDeadline(time.Time{}) // Clear deadline
	session, err := yamux.Server(conn, yamux.DefaultConfig())
	if err != nil {
		tt.logger.Errorf("Open session failed: %v", err)
		return false
	}
	tt.sesVer += 1 // Overflow?? no such thing..
	tt.session = session
	tt.logger.Debugf("Open session success")
	return true
}

func (tt *tcpBasedTunnel) getStream(retry int) (*yamux.Stream, error) {
	tt.RLock()
	session, version := tt.session, tt.sesVer
	tt.RUnlock()

	for i := 0; i < retry; i++ {
		if session == nil {
			// XXX(damnever): Update such status by client side?
			tt.tracker.OnError("local address may be not working")
			return nil, fmt.Errorf("local proxy has no activity")
		}
		stream, err := session.OpenStream() // May block here if too many packet in flight
		if err == nil {
			return stream, nil
		}
		if err == yamux.ErrStreamsExhausted {
			tt.tracker.OnError("too many open connections")
			return nil, err
		}
		session = tt.tryInvalidSession(version)
	}
	return nil, fmt.Errorf("max retry exceeded")
}

func (tt *tcpBasedTunnel) tryInvalidSession(version uint64) *yamux.Session {
	tt.Lock()
	defer tt.Unlock()
	if version == tt.sesVer {
		tt.session = nil
	}
	return tt.session
}

func (tt *tcpBasedTunnel) Serve() error {
	for {
		conn, err := tt.server.Accept()
		if err != nil {
			return err
		}

		tt.Add(1)
		go tt.handleConn(conn)
	}
}

func (tt *tcpBasedTunnel) handleConn(conn net.Conn) {
	defer tt.Done()
	defer func() {
		if e := recover(); e != nil {
			tt.logger.Panicf("Panic: %v", e)
		}
	}()
	tt.tracker.IncrConn()
	defer tt.tracker.DecrConn()

	stream, err := tt.getStream(2)
	if err != nil { // Write error message according to protocol
		tt.logger.Errorf("Open stream failed: %v", err)
		return
	}

	streamID := stream.StreamID()
	tt.logger.Infof("[%d] Linking stream: %s<->%s", streamID, stream.LocalAddr(), conn.LocalAddr())
	in, out := connutil.LinkStream(conn, stream)
	tt.tracker.RecordTraffic(in, out)
	tt.logger.Infof("[%d] Linked stream closed", streamID)
}

// Close closes the listener and set the closed flag,
// then no more new requests could be processed.
func (tt *tcpBasedTunnel) Close() {
	tt.Lock()
	if tt.closed {
		tt.Unlock()
		return
	}
	tt.closed = true
	tt.server.Close()
	tt.Unlock()

	tt.tracker.Closed()
}

// WaitAndCleanup wait in processing requests done then closes the session,
// since all streams will be closed if session closed.
func (tt *tcpBasedTunnel) WaitAndCleanup() {
	tt.Wait()

	tt.Lock()
	if tt.session != nil {
		tt.session.Close()
	}
	tt.Unlock()
}
