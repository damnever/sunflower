package flower

import (
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/damnever/sunflower/birpc"
	"github.com/damnever/sunflower/log"
	"github.com/damnever/sunflower/msg/msgpb"
	"github.com/damnever/sunflower/pkg/util"
)

type Controler struct {
	sync.RWMutex
	sync.WaitGroup

	conf    *Config
	client  *birpc.Client
	proxies map[string]*TCPProxy
	logger  *zap.SugaredLogger
}

func NewControler(conf *Config) (*Controler, error) {
	client, err := birpc.NewClient(conf.BuildRPCClientConf())
	if err != nil {
		return nil, err
	}
	return &Controler{
		conf:    conf,
		client:  client,
		proxies: map[string]*TCPProxy{},
		logger:  log.New("ctl[%s]", conf.Hash),
	}, nil
}

func (c *Controler) Run() error {
	errCh := make(chan error, 1)
	go func() { errCh <- c.client.Run(c) }()
	sigCh := util.WatchSignals()

	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
	case sig := <-sigCh:
		c.logger.Infof("Got signal: %v", sig)
	}

	c.client.Close()
	c.logger.Info("Graceful shutdown..")
	// graceful shutdown
	done := make(chan struct{})
	go func() {
		c.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(c.conf.Timeout.GracefulShutdown):
	}
	return nil
}

func (c *Controler) closeProxy(tunnelhash string) {
	c.Lock()
	if proxy, in := c.proxies[tunnelhash]; in {
		delete(c.proxies, tunnelhash)
		proxy.Close()
	}
	c.Unlock()
}

func (c *Controler) HandlePingResponse(req *msgpb.PingResponse) {
	c.logger.Debug("Received heartbeat response")
}

func (c *Controler) HandleNewTunnelRequest(req *msgpb.NewTunnelRequest) *msgpb.NewTunnelResponse {
	resp := &msgpb.NewTunnelResponse{TunnelHash: req.TunnelHash}
	if req.ID != c.conf.ID || req.ClientHash != c.conf.Hash {
		resp.ErrCode = msgpb.ErrCodeBadClient
		c.logger.Warnf("Bad new tunnel request: %+v", req)
		return resp
	}

	c.Lock()
	defer c.Unlock()

	if _, in := c.proxies[req.TunnelHash]; in {
		c.logger.Debugf("Tunnel %s already registered", req.TunnelHash)
		return resp
	}

	proxy, err := NewTCPProxy(req, c) // bad practice? fuck me..
	if err != nil {
		c.logger.Errorf("Failed to create local proxy(%8s): %s://%s", req.TunnelHash, req.Proto, req.ExportAddr)
		resp.ErrCode = msgpb.ErrCodeBadRegistryAddr
		return resp
	}

	c.proxies[req.TunnelHash] = proxy
	go func() {
		defer c.closeProxy(req.TunnelHash)
		proxy.Serve()
	}()
	return resp
}

func (c *Controler) HandleCloseTunnelRequest(req *msgpb.CloseTunnelRequest) *msgpb.CloseTunnelResponse {
	resp := &msgpb.CloseTunnelResponse{}
	if req.ID != c.conf.ID || req.ClientHash != c.conf.Hash {
		resp.ErrCode = msgpb.ErrCodeBadClient
		c.logger.Warnf("Bad close tunnel request: %+v", req)
		return resp
	}
	c.logger.Infof("Closing proxy: %8s", req.TunnelHash)
	c.closeProxy(req.TunnelHash)
	resp.TunnelHash = req.TunnelHash
	return resp
}

func (c *Controler) HandleShutdownRequest(req *msgpb.ShutdownRequest) bool {
	if req.ID != c.conf.ID || req.ClientHash != c.conf.Hash {
		c.logger.Warnf("Bad shutdown request: %+v", req)
		return false
	}
	c.logger.Infof("Shutdown request received")
	return true
}

func (c *Controler) HandleUnknownMessage(msg interface{}) {
	c.logger.Warnf("Received unknown message: %+v", msg)
}

func (c *Controler) HandleError(err error) {
	c.logger.Errorf("Error: %+v", err)
}
