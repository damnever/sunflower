package sun

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/damnever/sunflower/birpc"
	"github.com/damnever/sunflower/log"
	"github.com/damnever/sunflower/msg/msgpb"
	"github.com/damnever/sunflower/sun/pubsub"
	"github.com/damnever/sunflower/sun/registry"
	"github.com/damnever/sunflower/sun/storage"
	"github.com/damnever/sunflower/sun/tracker"
	"github.com/damnever/sunflower/version"
)

type CtlServer struct {
	sync.Mutex
	logger           *zap.SugaredLogger
	server           *birpc.Server
	db               *storage.DB
	reg              *registry.TCPTunnelRegistry
	sub              pubsub.Subscriber
	filter           map[string]bool
	done             chan struct{}
	tracker          *tracker.Tracker
	gracefulShutdown time.Duration
}

func NewCtlServer(conf Config, sub pubsub.Subscriber, db *storage.DB) (*CtlServer, error) {
	reg, err := registry.New(conf.MuxRegConf)
	if err != nil {
		return nil, err
	}

	s := &CtlServer{
		logger:           log.New("S"),
		db:               db,
		reg:              reg,
		sub:              sub,
		filter:           map[string]bool{},
		tracker:          tracker.New(db),
		done:             make(chan struct{}),
		gracefulShutdown: conf.GracefulShutdown,
	}

	conf.RPCConf.ValidateFunc = s.ValidateClient
	server, err := birpc.NewServer(&conf.RPCConf)
	if err != nil {
		return nil, err
	}
	s.server = server
	return s, nil
}

func (s *CtlServer) ValidateClient(id, hash, device, ver string) msgpb.ErrCode {
	s.Lock()
	defer s.Unlock()

	key := fmt.Sprintf("%s:%s", id, hash)
	if s.filter[key] {
		s.logger.Warnf("Agent (%s %s) already connected", id, hash)
		return msgpb.ErrCodeDuplicateAgent
	}

	if !version.IsCompatible(ver) {
		return msgpb.ErrCodeBadVersion
	}

	ok, err := s.db.UpdateAgent(
		id, hash,
		map[string]interface{}{"device": device, "version": ver},
	)
	if err != nil {
		s.logger.Errorf("Update agent (%s %s) failed: %v", id, hash, err)
		return msgpb.ErrCodeInternalServerError
	}
	if !ok {
		s.logger.Warnf("Agent (%s %s) may not exists", id, hash)
		return msgpb.ErrCodeBadClient
	}
	s.filter[key] = true
	return msgpb.ErrCodeNull
}

func (s *CtlServer) removeFilter(id, hash string) {
	s.Lock()
	delete(s.filter, fmt.Sprintf("%s:%s", id, hash))
	s.Unlock()
}

func (s *CtlServer) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 2)
	// Tunnel registry
	go func() { errCh <- s.reg.Serve() }()
	defer s.reg.Close()
	// Control server
	go func() { errCh <- s.server.Serve() }()
	defer s.server.Close()

	for {
		select {
		case <-s.done:
			return nil
		case err := <-errCh:
			return err
		case conn := <-s.server.Clients():
			s.logger.Infof("New client come in: %s(%s)", conn.ID, conn.Hash)
			cli := &CtlClient{
				ClientConn: conn,
				tracker:    s.tracker.AgentTracker(conn.ID, conn.Hash),
				reg:        s.reg,
				sub:        s.sub,
				db:         s.db,
				logger:     log.New("ctl[%s]", conn.Hash),
			}
			go func() {
				defer s.removeFilter(conn.ID, conn.Hash)
				cli.Run(ctx)
			}()
		}
	}
}

func (s *CtlServer) GracefulShutdown() {
	close(s.done)

	s.logger.Info("Graceful shutdown..")
	done := make(chan struct{})
	go func() {
		s.reg.WaitDone()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(s.gracefulShutdown):
	}
}
