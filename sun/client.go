package sun

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/damnever/sunflower/birpc"
	"github.com/damnever/sunflower/msg"
	"github.com/damnever/sunflower/msg/msgpb"
	"github.com/damnever/sunflower/pkg/delaytimer"
	"github.com/damnever/sunflower/pkg/util"
	"github.com/damnever/sunflower/sun/pubsub"
	"github.com/damnever/sunflower/sun/registry"
	"github.com/damnever/sunflower/sun/storage"
	"github.com/damnever/sunflower/sun/tracker"
)

const (
	defaultDelayTimerBufferSize = 10
)

type CtlClient struct {
	*birpc.ClientConn
	reg        *registry.TCPTunnelRegistry
	sub        pubsub.Subscriber
	db         *storage.DB
	tracker    *tracker.AgentTracker
	logger     *zap.SugaredLogger
	lastPingT  time.Time
	delayTimer *delaytimer.DelayTimer
}

func (c *CtlClient) Run(ctx context.Context) {
	defer func() {
		if e := recover(); e != nil {
			c.logger.Errorf("Stoped with error: %v", e)
		}
	}()
	defer c.tracker.Disconnected()
	defer c.reg.DeregisterAll(c.Hash)

	c.tracker.Connected()
	util.Must(c.openAllTunnels())

	c.Go()
	defer c.Close()
	evtCh := c.sub.Sub(c.Hash)
	defer c.sub.Unsub(c.Hash)

PROCESS_LOOP:
	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Stopping..")
			break PROCESS_LOOP
		case err := <-c.Err():
			c.logger.Errorf("Connection error: %v", err)
			break PROCESS_LOOP
		case evt := <-evtCh:
			util.Must(c.handleEvent(evt))
		case m := <-c.In():
			c.handleMessage(m)
		}
	}
}

func (c *CtlClient) handleMessage(m interface{}) {
	switch x := m.(type) {
	case *msgpb.PingRequest:
		now := time.Now()
		// Response first, so we can calculate delay
		// during the gap of ping-pong cycle.
		c.Out() <- msgpb.PingResponse{}

		if c.lastPingT.IsZero() {
			c.delayTimer = delaytimer.New(defaultDelayTimerBufferSize)
		} else {
			d := now.Sub(c.lastPingT)
			delayed := c.delayTimer.Calc(d)
			c.tracker.Delayed(delayed)
		}
		c.lastPingT = now
	case *msgpb.NewTunnelResponse:
		// XXX(damnever): create tunnel after response received?
		// since we deregister all tunnels when connection disrupted,
		// so leave it that way for now.
		c.logger.Debugf("NewTunnelResponse: %+v", x)
		if err := msg.CodeToError(x.ErrCode); err != nil {
			c.logger.Errorf("Received bad NewTunnelResponse: %v", err)
			c.reg.Deregister(c.Hash, x.TunnelHash)
		} else {
			c.logger.Infof("Tunnel %s registered", x.TunnelHash)
		}
	case *msgpb.CloseTunnelResponse:
		if err := msg.CodeToError(x.ErrCode); err != nil {
			c.logger.Errorf("Received bad CloseTunnelResponse: %v", err)
		} else {
			c.logger.Infof("Tunnel %s has been closed", x.TunnelHash)
		}
		c.logger.Debugf("CloseTunnelResponse: %+v", x)
	default:
		c.logger.Warnf("Unknown message: %+v", x)
	}
}

func (c *CtlClient) handleEvent(evt *pubsub.Event) error {
	id, ahash, thash := c.ID, c.Hash, evt.TunnelHash

	switch evt.Type {
	case pubsub.EventOpenTunnel:
		tunnel, err := c.db.QueryTunnel(id, ahash, thash)
		if err != nil {
			return err
		}
		c.openTunnel(tunnel)
	case pubsub.EventCloseTunnel:
		if !c.reg.Deregister(ahash, thash) {
			return nil
		}

		c.Out() <- &msgpb.CloseTunnelRequest{
			ID:         id,
			ClientHash: ahash,
			TunnelHash: thash,
		}
	case pubsub.EventRejectAgent:
		c.Out() <- &msgpb.ShutdownRequest{}
		// XXX(damnever): better method to ensure message has been send.
		time.Sleep(1 * time.Second)
		return fmt.Errorf("reject self")
	}
	return nil
}

func (c *CtlClient) openAllTunnels() error {
	tunnels, err := c.db.QueryTunnels(c.ID, c.Hash)
	if err != nil {
		return err
	}
	for _, tunnel := range tunnels { // only 10 tunnels allowed per agent
		if !tunnel.Enabled {
			continue
		}
		c.openTunnel(tunnel)
	}
	return nil
}

func (c *CtlClient) openTunnel(tunnel storage.Tunnel) {
	err := c.reg.Register(
		c.tracker.TunnelTracker(tunnel.Hash),
		tunnel.Proto, tunnel.ServerAddr,
	)
	if err != nil {
		c.logger.Errorf("Open tunnel %s failed: %v", tunnel.Hash, err)
		return
	}

	c.Out() <- &msgpb.NewTunnelRequest{
		ID:           c.ID,
		ClientHash:   c.Hash,
		TunnelHash:   tunnel.Hash,
		Proto:        tunnel.Proto,
		ExportAddr:   tunnel.ExportAddr,
		RegistryAddr: c.reg.ListenAddr(),
	}
}
