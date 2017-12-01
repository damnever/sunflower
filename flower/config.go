package flower

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	zipexe "github.com/daaku/go.zipexe"
	"github.com/damnever/cc"
	"github.com/kardianos/osext"

	"github.com/damnever/sunflower/birpc"
	"github.com/damnever/sunflower/pkg/retry"
	"github.com/damnever/sunflower/pkg/util"
)

type Config struct {
	ID                string
	Hash              string
	ControlServer     string
	HeartbeatInterval time.Duration
	Timeout           struct {
		GracefulShutdown time.Duration
		Control          util.TimeoutConfig
		Tunnel           util.TimeoutConfig
		Local            util.TimeoutConfig
	}
	Retrier *retry.Retrier
}

func loadConfigFromExec() ([]byte, error) {
	exPath, err := osext.Executable()
	if err != nil {
		return nil, err
	}
	cr, zr, err := zipexe.OpenCloser(exPath)
	if err != nil {
		return nil, err
	}
	defer cr.Close()
	for _, f := range zr.File {
		if strings.HasSuffix(f.Name, ".yaml") {
			r, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer r.Close()
			return ioutil.ReadAll(r)
		}
	}
	return nil, fmt.Errorf("no config found from: %v", exPath)
}

func buildConfig(rawConf cc.Configer) *Config {
	conf := &Config{}
	conf.ID = rawConf.String("id")
	conf.Hash = rawConf.String("hash")
	conf.ControlServer = rawConf.String("control_server")
	conf.HeartbeatInterval = rawConf.DurationAndOr("heartbeat_interval", "N>=3", 3) * time.Second

	retryC := rawConf.Config("retry")
	backoff := retryC.DurationAndOr("backoff", "N>=100", 500) * time.Millisecond
	max := retryC.IntAndOr("max", "N>=3", 10)
	conf.Retrier = retry.New(backoff, max)

	timeoutC := rawConf.Config("timeout")
	conf.Timeout.GracefulShutdown = timeoutC.DurationAndOr("graceful_timeout", "N>0", 3) * time.Second

	controlC := timeoutC.Config("remote")
	readVP := fmt.Sprintf("N>%d&&N>=3000", int(conf.HeartbeatInterval.Seconds()*float64(1000)))
	conf.Timeout.Control = util.TimeoutConfig{
		Connect: controlC.DurationAndOr("connect", "N>=500", 5000) * time.Millisecond,
		Read:    controlC.DurationAndOr("read", readVP, 10000) * time.Millisecond,
		Write:   controlC.DurationAndOr("write", "N>0", 500) * time.Millisecond,
	}

	tunnelC := timeoutC.Config("tunnel")
	conf.Timeout.Tunnel = util.TimeoutConfig{
		Connect: tunnelC.DurationAndOr("connect", "N>=100", 2000) * time.Millisecond,
		Read:    tunnelC.DurationAndOr("read", "N>=100", 2000) * time.Millisecond,
		Write:   tunnelC.DurationAndOr("write", "N>0", 300) * time.Millisecond,
	}

	localC := timeoutC.Config("local")
	conf.Timeout.Local = util.TimeoutConfig{
		Connect: localC.DurationAndOr("connect", "N>=10", 500) * time.Millisecond,
		Read:    localC.DurationAndOr("connect", "N>=10", 500) * time.Millisecond,
		Write:   localC.DurationAndOr("connect", "N>0", 100) * time.Millisecond,
	}

	return conf
}

func (conf *Config) BuildRPCClientConf() *birpc.ClientConfig {
	rpcconf := &birpc.ClientConfig{}
	rpcconf.ID = conf.ID
	rpcconf.Hash = conf.Hash
	rpcconf.RemoteAddr = conf.ControlServer
	rpcconf.HeartbeatInterval = conf.HeartbeatInterval
	rpcconf.Retrier = conf.Retrier
	rpcconf.Timeout = conf.Timeout.Control
	rpcconf.TLSConf = &tls.Config{}
	return rpcconf
}
