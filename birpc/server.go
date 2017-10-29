package birpc

import (
	"crypto/tls"
	"net"
	"time"

	"github.com/damnever/sunflower/msg"
	"github.com/damnever/sunflower/msg/msgpb"
	"github.com/damnever/sunflower/pkg/util"
)

type ValidateFunc func(id, hash, device, version string) msgpb.ErrCode

type ServerConfig struct {
	ListenAddr   string
	Timeout      util.TimeoutConfig
	TLSConf      *tls.Config
	ValidateFunc ValidateFunc
}

type Server struct {
	l      net.Listener
	config *ServerConfig
	closed chan struct{}
	cliCh  chan *ClientConn
}

func NewServer(conf *ServerConfig) (*Server, error) {
	l, err := net.Listen("tcp", conf.ListenAddr)
	// l, err := tls.Listen("tcp", conf.ListenAddr, conf.TLSConf)
	if err != nil {
		return nil, err
	}
	return &Server{
		l:      l,
		config: conf,
		closed: make(chan struct{}),
		cliCh:  make(chan *ClientConn, 128),
	}, nil
}

func (s *Server) Serve() error {
	defer close(s.cliCh)

	for {
		conn, err := s.l.Accept()
		if err != nil {
			select {
			case <-s.closed:
				return nil
			default:
			}
			return err
		}
		go s.handleConn(conn)
	}
}

func (s *Server) Clients() <-chan *ClientConn {
	return s.cliCh
}

func (s *Server) handleConn(conn net.Conn) {
	conf := s.config
	conn.SetReadDeadline(time.Now().Add(conf.Timeout.Read))
	var req msgpb.HandshakeRequest
	if err := msg.ReadTo(conn, &req); err != nil {
		conn.Close()
		return
	}

	resp := msgpb.HandshakeResponse{
		ErrCode: conf.ValidateFunc(req.ID, req.Hash, req.Device, req.Version),
	}

	conn.SetReadDeadline(time.Now().Add(conf.Timeout.Write))
	if err := msg.Write(conn, &resp); err != nil {
		conn.Close()
		return
	}
	if resp.ErrCode != msgpb.ErrCodeNull {
		conn.Close()
		return
	}

	s.cliCh <- &ClientConn{
		Conn: NewConn(conn, conf.Timeout.Read, conf.Timeout.Write),
		ID:   req.ID,
		Hash: req.Hash,
	}
}

func (s *Server) Close() error {
	close(s.closed)
	return s.l.Close()
}

type ClientConn struct {
	*Conn
	ID      string
	Hash    string
	Version string
	Device  string
}
