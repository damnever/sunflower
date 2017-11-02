package birpc

import (
	"crypto/tls"
	"fmt"
	"net"
	"runtime"
	"time"

	"github.com/damnever/sunflower/msg"
	"github.com/damnever/sunflower/msg/msgpb"
	"github.com/damnever/sunflower/pkg/retry"
	"github.com/damnever/sunflower/pkg/util"
	"github.com/damnever/sunflower/version"
)

// TODO(damnever):
//  - more details about device information
//  - report client side stats during heartbeat
//    - runtime: goroutines, trace, profile, gc, etc.

var deviceInfo = fmt.Sprintf("%v/%v", runtime.GOOS, runtime.GOARCH)

type ClientHandler interface {
	HandlePingResponse(req *msgpb.PingResponse)
	HandleNewTunnelRequest(req *msgpb.NewTunnelRequest) *msgpb.NewTunnelResponse
	HandleCloseTunnelRequest(req *msgpb.CloseTunnelRequest) *msgpb.CloseTunnelResponse
	HandleShutdownRequest(req *msgpb.ShutdownRequest) bool
	HandleUnknownMessage(m interface{})
	HandleError(err error)
}

type ClientConfig struct {
	ID                string
	Hash              string
	RemoteAddr        string
	HeartbeatInterval time.Duration
	Timeout           util.TimeoutConfig
	Retrier           *retry.Retrier
	TLSConf           *tls.Config
}

type Client struct {
	config *ClientConfig
	conn   net.Conn
	closed chan struct{}
}

func NewClient(conf *ClientConfig) (*Client, error) {
	conn, err := connectAndHandshake(conf)
	if err != nil {
		return nil, err
	}
	return &Client{
		config: conf,
		conn:   conn,
		closed: make(chan struct{}),
	}, nil
}

// Run starts communicating with server, do reconnecting and requests dispatching logic.
func (cli *Client) Run(handler ClientHandler) error {
	pingReq := &msgpb.PingRequest{}

	conf := cli.config
	conn := NewConn(cli.conn, conf.Timeout.Read, conf.Timeout.Write)
	conn.Go()
	defer func() { conn.Close() }()

	ticker := time.NewTicker(conf.HeartbeatInterval)
	defer ticker.Stop()
	tryConnect := tryConnectFunc(conf, cli.closed)

LOOP:
	for {
		select {
		case <-cli.closed:
			return nil
		case err := <-conn.Err():
			if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
				continue LOOP
			}
			handler.HandleError(err)
			conn.Close()
			rawConn, fatalErr := tryConnect()
			if fatalErr != nil {
				if fatalErr == retry.ErrCanceled {
					return nil
				}
				return fatalErr
			}
			conn = NewConn(rawConn, conf.Timeout.Read, conf.Timeout.Write)
			conn.Go()

		case m := <-conn.In():
			switch x := m.(type) {
			case *msgpb.PingResponse:
				go handler.HandlePingResponse(x)
			case *msgpb.NewTunnelRequest:
				go func() { conn.Out() <- handler.HandleNewTunnelRequest(x) }()
			case *msgpb.CloseTunnelRequest:
				go func() { conn.Out() <- handler.HandleCloseTunnelRequest(x) }()
			case *msgpb.ShutdownRequest:
				if handler.HandleShutdownRequest(x) {
					return nil
				}
			default:
				go handler.HandleUnknownMessage(x)
			}

		case <-ticker.C:
			conn.Out() <- pingReq
		}
	}
}

func (cli *Client) Close() error {
	close(cli.closed)
	return cli.conn.Close()
}

func tryConnectFunc(conf *ClientConfig, closed chan struct{}) func() (net.Conn, error) {
	return func() (conn net.Conn, err error) {
		conf.Retrier.Run(func() error {
			select {
			case <-closed:
				err = retry.ErrCanceled
				return err
			default:
			}
			conn, err = connectAndHandshake(conf)
			return err
		})
		return
	}
}

func connectAndHandshake(conf *ClientConfig) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", conf.RemoteAddr, conf.Timeout.Connect)
	if err != nil {
		return nil, err
	}

	req := &msgpb.HandshakeRequest{
		ID:      conf.ID,
		Hash:    conf.Hash,
		Version: version.Info(),
		Device:  deviceInfo,
	}
	conn.SetWriteDeadline(time.Now().Add(conf.Timeout.Write))
	if err := msg.Write(conn, req); err != nil {
		return nil, err
	}

	conn.SetReadDeadline(time.Now().Add(conf.Timeout.Read))
	var resp msgpb.HandshakeResponse
	if err = msg.ReadTo(conn, &resp); err != nil {
		return nil, err
	}
	if err = msg.CodeToError(resp.ErrCode); err != nil {
		return nil, err
	}
	return conn, nil
}
