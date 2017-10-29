package registry

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/damnever/sunflower/log"
	"github.com/damnever/sunflower/pkg/bufpool"
	"github.com/damnever/sunflower/pkg/util"
)

var (
	errClosed             = fmt.Errorf("listener already closed")
	httpConnAcceptTimeout = 100 * time.Millisecond
	noSuchTunnel          = "%s 404 Not Found\r\nContent-Length: %d\r\n\r\n%s\r\n"
)

type HTTPTunnelMuxer struct {
	sync.RWMutex
	logger   *zap.SugaredLogger
	domain   string
	l        net.Listener
	registry map[string]*httpConnListener
	closed   bool
}

func NewHTTPTunnelMuxer(domain string, addr string) (*HTTPTunnelMuxer, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &HTTPTunnelMuxer{
		logger:   log.New("mux[http]"),
		l:        l,
		domain:   fmt.Sprintf(".%s", domain),
		registry: map[string]*httpConnListener{},
		closed:   false,
	}, nil
}

func (hm *HTTPTunnelMuxer) Serve() error {
	defer hm.cleanup()
	for {
		conn, err := hm.l.Accept()
		if err != nil {
			return err
		}
		go hm.handleConn(conn)
	}
}

func (hm *HTTPTunnelMuxer) handleConn(conn net.Conn) {
	defer func() {
		if e := recover(); e != nil {
			hm.logger.Panicf("Panic: %v", e)
		}
	}()

	hc, rd := newHTTPConn(conn)
	req, err := http.ReadRequest(bufio.NewReader(rd))
	if err != nil {
		if err != io.EOF { // Too many noise
			hm.logger.Errorf("Failed to read request: %v", err)
		}
		hc.Close()
		return
	}
	defer req.Body.Close()

	subdomain := strings.TrimSuffix(util.Host(req), hm.domain)

	hm.RLock()
	hl, in := hm.registry[subdomain]
	hm.RUnlock()
	if in {
		select {
		case hl.connCh <- hc:
		case <-time.After(httpConnAcceptTimeout):
			hc.Close()
		}
		return
	}

	defer hc.Close()
	msg := fmt.Sprintf("No such tunnel: %s", subdomain)
	content := fmt.Sprintf(noSuchTunnel, req.Proto, len(msg), msg)
	if _, err := conn.Write([]byte(content)); err != nil {
		hm.logger.Errorf("Failed to write error response: %v", err)
	}
}

func (hm *HTTPTunnelMuxer) Listen(subdomain string) (*httpConnListener, error) {
	hm.Lock()
	defer hm.Unlock()
	if hm.closed {
		return nil, errClosed
	}
	if hl, in := hm.registry[subdomain]; in {
		return hl, nil
	}
	hl := newHTTPConnListener(subdomain, hm)
	hm.registry[subdomain] = hl
	return hl, nil
}

func (hm *HTTPTunnelMuxer) unListen(subdomain string) {
	hm.Lock()
	delete(hm.registry, subdomain)
	hm.Unlock()
}

func (hm *HTTPTunnelMuxer) cleanup() {
	hm.Lock()
	defer hm.Unlock()
	if hm.closed {
		return
	}
	hm.closed = true
	for _, hl := range hm.registry {
		hl.Close()
	}
}

func (hm *HTTPTunnelMuxer) Close() error {
	return hm.l.Close()
}

type httpConnListener struct {
	subdomain string
	muxer     *HTTPTunnelMuxer
	connCh    chan *httpConn
	errCh     chan error
	done      chan struct{}
}

func newHTTPConnListener(subdomain string, muxer *HTTPTunnelMuxer) *httpConnListener {
	return &httpConnListener{
		subdomain: subdomain,
		muxer:     muxer,
		connCh:    make(chan *httpConn, 16),
		done:      make(chan struct{}),
	}
}

func (hl *httpConnListener) Accept() (net.Conn, error) {
	select {
	case <-hl.done:
		go hl.cleanup()
		return nil, errClosed
	case conn := <-hl.connCh:
		return conn, nil
	}
}

func (hl *httpConnListener) cleanup() {
	// FIXME(damnever): what the fuck is this shit?
	time.Sleep(httpConnAcceptTimeout)
	close(hl.connCh)
	for hc := range hl.connCh {
		hc.Close() // release buf
	}
}

func (hl *httpConnListener) Addr() net.Addr {
	return hl.muxer.l.Addr()
}

func (hl *httpConnListener) Close() error {
	hl.muxer.unListen(hl.subdomain)
	close(hl.done)
	return nil
}

type httpConn struct {
	net.Conn
	mu  sync.Mutex
	buf *bytes.Buffer
}

func newHTTPConn(conn net.Conn) (*httpConn, io.Reader) {
	buf := bufpool.Get()
	return &httpConn{
		Conn: conn,
		buf:  buf,
	}, io.TeeReader(conn, buf)
}

func (hc *httpConn) Read(p []byte) (int, error) {
	if hc.buf == nil {
		return hc.Conn.Read(p)
	}
	n, err := hc.buf.Read(p)
	if err == io.EOF {
		hc.releaseBuf()
		var n2 int
		n2, err = hc.Conn.Read(p[n:])
		n += n2
	}
	return n, err
}

func (hc *httpConn) Close() error {
	hc.releaseBuf()
	return hc.Conn.Close()
}

func (hc *httpConn) releaseBuf() {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	if hc.buf == nil {
		return
	}
	bufpool.Put(hc.buf)
	hc.buf = nil
}
