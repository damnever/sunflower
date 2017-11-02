package flower

import (
	"net"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/yamux"
	"go.uber.org/zap"

	"github.com/damnever/sunflower/log"
	"github.com/damnever/sunflower/msg"
	"github.com/damnever/sunflower/msg/msgpb"
	connutil "github.com/damnever/sunflower/pkg/conn"
	"github.com/damnever/sunflower/pkg/retry"
)

// TODO(damnever): refactor..

const (
	localConnectTimeout = 1 * time.Second
)

type registerFunc func() (*yamux.Session, error)

type TCPProxy struct {
	sync.Mutex
	ctl        *Controler
	regSelf    registerFunc
	logger     *zap.SugaredLogger
	exportAddr string
	session    *yamux.Session
	closed     bool
}

func NewTCPProxy(req *msgpb.NewTunnelRequest, ctl *Controler) (*TCPProxy, error) {
	logger := log.New("prx[%s://%s]", strings.ToLower(req.Proto), req.ExportAddr)
	p := &TCPProxy{
		ctl:        ctl,
		logger:     logger,
		exportAddr: req.ExportAddr,
		closed:     false,
	}

	regSelf := p.tryRegisterProxyFunc(req, ctl.conf)
	session, err := regSelf()
	if err != nil {
		return nil, err
	}
	p.session = session
	p.regSelf = regSelf
	return p, nil
}

func (p *TCPProxy) Close() error {
	p.Lock()
	defer p.Unlock()
	if p.closed {
		return nil
	}
	p.closed = true
	return p.session.Close()
}

func (p *TCPProxy) isclosed() bool {
	p.Lock()
	defer p.Unlock()
	return p.closed
}

func (p *TCPProxy) Serve() {
	for {
		stream, err := p.session.AcceptStream()
		if err == nil {
			p.ctl.Add(1)
			go p.handleStream(stream)
			continue
		}
		if err == yamux.ErrTimeout || err == yamux.ErrStreamsExhausted {
			p.logger.Warnf("Accept stream: %v", err)
			continue
		}
		if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
			continue
		}

		if p.isclosed() {
			break
		}
		p.logger.Errorf("Got error: %v, try reconnecting..", err)

		sess, fatalErr := p.regSelf()
		if fatalErr != nil {
			if fatalErr != retry.ErrCanceled {
				p.logger.Errorf("Reconnect failed: %v", fatalErr)
			}
			break
		}
		p.Lock()
		if p.closed {
			p.Unlock()
			sess.Close()
			break
		}
		p.session = sess
		p.Unlock()
	}

	p.logger.Info("Stopped")
}

func (p *TCPProxy) handleStream(stream *yamux.Stream) {
	defer p.ctl.Done()
	streamID := stream.StreamID()
	defer func() {
		if e := recover(); e != nil {
			p.logger.Panicf("[%d] Panic: %v", streamID, e)
		}
	}()

	timeout := p.ctl.conf.Timeout.Local.Connect
	localConn, err := net.DialTimeout("tcp", p.exportAddr, timeout)
	if err != nil {
		stream.Close()
		p.logger.Errorf("[%v] Failed to connect to %v: %v", streamID, p.exportAddr, err)
		return
	}

	p.logger.Infof("[%v] Linking stream: %v<->%v", streamID, localConn.RemoteAddr(), stream.RemoteAddr())
	connutil.LinkStream(stream, localConn)
	p.logger.Infof("[%v] Linked stream closed", streamID)
}

func (p *TCPProxy) tryRegisterProxyFunc(req *msgpb.NewTunnelRequest, conf *Config) registerFunc {
	cliID := req.ID
	cliHash := req.ClientHash
	tunnelHash := req.TunnelHash
	registryAddr := req.RegistryAddr
	retrier := conf.Retrier
	timeout := conf.Timeout.Tunnel

	return func() (session *yamux.Session, err error) {
		retrier.Run(func() error {
			if p.isclosed() {
				err = retry.ErrCanceled // the error..
				return err
			}

			var conn net.Conn
			conn, err = net.DialTimeout("tcp", registryAddr, timeout.Connect)
			if err != nil {
				p.logger.Errorf("Connect to registry failed: %v", err)
				return err
			}

			err = doHandshake(conn, timeout.Read, timeout.Write, &msgpb.TunnelHandshakeRequest{
				ID:         cliID,
				ClientHash: cliHash,
				TunnelHash: tunnelHash,
			})
			if err != nil {
				conn.Close()
				p.logger.Errorf("Handshake failed: %v", err)
				return err
			}

			conn.SetDeadline(time.Time{}) // Clear deadline
			if session, err = yamux.Client(conn, yamux.DefaultConfig()); err != nil {
				conn.Close()
				p.logger.Errorf("Create session failed: %v", err)
				return err
			}
			return nil
		})
		return
	}
}

func doHandshake(conn net.Conn, rTimeout, wTimeout time.Duration, req *msgpb.TunnelHandshakeRequest) error {
	conn.SetWriteDeadline(time.Now().Add(wTimeout))
	if err := msg.Write(conn, req); err != nil {
		return err
	}
	var resp msgpb.TunnelHandshakeResponse
	conn.SetReadDeadline(time.Now().Add(rTimeout))
	if err := msg.ReadTo(conn, &resp); err != nil {
		return err
	}
	return msg.CodeToError(resp.ErrCode)
}
