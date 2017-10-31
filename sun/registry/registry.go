package registry

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/damnever/sunflower/log"
	"github.com/damnever/sunflower/msg"
	"github.com/damnever/sunflower/msg/msgpb"
	"github.com/damnever/sunflower/pkg/util"
	"github.com/damnever/sunflower/sun/tracker"
)

type Config struct {
	IP       string
	Domain   string
	HTTPAddr string
	Timeout  util.TimeoutConfig
}

type TCPTunnelRegistry struct {
	sync.RWMutex
	sync.WaitGroup

	logger    *zap.SugaredLogger
	timeout   util.TimeoutConfig
	tunneln   net.Listener
	tlnAddr   string
	httpmuxer *HTTPTunnelMuxer
	tunnels   map[string]map[string]Tunnel
}

func New(conf Config) (*TCPTunnelRegistry, error) {
	ln, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		return nil, err
	}

	var muxer *HTTPTunnelMuxer
	if conf.Domain != "" {
		if muxer, err = NewHTTPTunnelMuxer(conf.Domain, conf.HTTPAddr); err != nil {
			ln.Close()
			return nil, err
		}
	}

	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		ln.Close()
		if muxer != nil {
			muxer.Close()
		}
		return nil, err
	}
	lnAddr := fmt.Sprintf("%s:%s", conf.IP, port)
	return &TCPTunnelRegistry{
		timeout:   conf.Timeout,
		logger:    log.New("reg[tcp]"),
		tunneln:   ln,
		tlnAddr:   lnAddr,
		httpmuxer: muxer,
		tunnels:   map[string]map[string]Tunnel{},
	}, nil
}

func (tr *TCPTunnelRegistry) ListenAddr() string {
	return tr.tlnAddr
}

func (tr *TCPTunnelRegistry) Serve() error {
	errCh := make(chan error, 2)
	go func() { errCh <- tr.serveHTTPMuxer() }()
	go func() { errCh <- tr.serveIncomingTunnel() }()
	return <-errCh
}

func (tr *TCPTunnelRegistry) serveHTTPMuxer() error {
	return tr.httpmuxer.Serve()
}

func (tr *TCPTunnelRegistry) serveIncomingTunnel() error {
	for {
		conn, err := tr.tunneln.Accept()
		if err != nil {
			return err
		}
		go tr.handleIncomingTunnelConn(conn)
	}
}

func (tr *TCPTunnelRegistry) handleIncomingTunnelConn(conn net.Conn) {
	// One connection per tunnel, if server side found duplicate
	// connection from client, simply close the connection
	var req msgpb.TunnelHandshakeRequest
	conn.SetReadDeadline(time.Now().Add(tr.timeout.Read))
	if err := msg.ReadTo(conn, &req); err != nil {
		tr.logger.Warnf("Read handshake request failed: %v", err)
		conn.Close()
		return
	}

	resp := msgpb.TunnelHandshakeResponse{}
	tr.RLock()
	var tunnel Tunnel
	if cliTunnels, in := tr.tunnels[req.ClientHash]; in {
		tunnel = cliTunnels[req.TunnelHash]
	}
	tr.RUnlock()

	if tunnel == nil {
		tr.logger.Infof("No tunnel registered for: <%s:%s>", req.ClientHash, req.TunnelHash)
		resp.ErrCode = msgpb.ErrCodeNoSuchTunnel
	}

	conn.SetWriteDeadline(time.Now().Add(tr.timeout.Write))
	if err := msg.Write(conn, resp); err != nil {
		tr.logger.Warnf("Write handshake response to <%s:%s> failed: %v", req.ClientHash, req.TunnelHash, err)
		conn.Close()
		return
	}

	conn.SetDeadline(time.Time{})
	if tunnel != nil && !tunnel.NewSession(conn) {
		tr.logger.Infof("Dumplicate connection for: <%s:%s>", req.ClientHash, req.TunnelHash)
	}
}

func (tr *TCPTunnelRegistry) Register(tracker *tracker.TunnelTracker, proto, serverAddr string) error {
	ahash, thash := tracker.AgentHash(), tracker.Hash()
	tr.Lock()
	defer tr.Unlock()

	etunnels, in := tr.tunnels[ahash]
	if !in {
		etunnels = make(map[string]Tunnel, 5)
		tr.tunnels[ahash] = etunnels
	}
	if _, in := etunnels[thash]; in {
		return nil
	}

	tunnel, err := tr.makeTunnel(tracker, proto, serverAddr)
	if err != nil {
		return err
	}

	tr.Add(1)
	go func() {
		defer tr.Done()
		defer tunnel.WaitAndCleanup()
		defer tr.Deregister(ahash, thash)
		tunnel.Serve()
	}()

	etunnels[thash] = tunnel
	tr.logger.Infof("New tunnel <%8s:%8s> registered", ahash, thash)
	return nil
}

func (tr *TCPTunnelRegistry) makeTunnel(tracker *tracker.TunnelTracker, proto, serverAddr string) (Tunnel, error) {
	var (
		tunnel Tunnel
		err    error
	)
	// TODO(damnever): check the subdomain or listen address whether is legal
	proto = strings.ToLower(proto)
	switch proto {
	case "http", "tcp":
		if proto == "http" && tr.httpmuxer != nil {
			var l net.Listener
			if l, err = tr.httpmuxer.Listen(serverAddr); err == nil {
				tunnel = NewHTTPTunnel(tracker, l)
			}
		} else {
			tunnel, err = NewTCPTunnel(tracker, serverAddr)
		}
	default:
		err = fmt.Errorf("Unsupported protocol: %s", proto)
	}
	return tunnel, err
}

func (tr *TCPTunnelRegistry) Deregister(ahash, thash string) bool {
	tr.Lock()
	defer tr.Unlock()

	if etunnels, in := tr.tunnels[ahash]; in {
		if tunnel, in := etunnels[thash]; in {
			tr.logger.Infof("Tunnel <%8s:%8s> deregistered", ahash, thash)
			tunnel.Close()
			delete(etunnels, thash)
			return true
		}
	}
	return false
}

func (tr *TCPTunnelRegistry) DeregisterAll(ahash string) bool {
	tr.Lock()
	defer tr.Unlock()

	etunnels, in := tr.tunnels[ahash]
	if !in {
		return false
	}

	for thash, tunnel := range etunnels {
		tr.logger.Infof("Tunnel <%s:%s> deregistered", ahash, thash)
		tunnel.Close()
	}
	delete(tr.tunnels, ahash)
	return true
}

func (tr *TCPTunnelRegistry) Close() {
	tr.Lock()
	tr.tunneln.Close()
	if tr.httpmuxer != nil {
		tr.httpmuxer.Close()
	}

	for _, etunnels := range tr.tunnels {
		for _, tunnel := range etunnels {
			tunnel.Close()
		}
	}
	tr.Unlock()
}

func (tr *TCPTunnelRegistry) WaitDone() {
	tr.Wait()
}
