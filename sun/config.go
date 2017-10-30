package sun

import (
	"crypto/tls"
	"time"

	"github.com/damnever/cc"

	"github.com/damnever/sunflower/birpc"
	"github.com/damnever/sunflower/pkg/util"
	"github.com/damnever/sunflower/sun/web"
)

type Config struct {
	ControlAddr string
	HTTP        struct {
		Domain string
		Addr   string
	}
	Timeout struct {
		GracefulShutdown time.Duration
		Control          util.TimeoutConfig
		Tunnel           util.TimeoutConfig
	}
}

func BuildConfig(rawConf cc.Configer) *Config {
	conf := &Config{}
	conf.ControlAddr = rawConf.String("addr")

	httpC := rawConf.Config("http")
	conf.HTTP.Domain = httpC.String("domain")
	conf.HTTP.Addr = httpC.String("addr")

	timeoutC := rawConf.Config("timeout")
	conf.Timeout.GracefulShutdown = timeoutC.DurationAndOr("graceful_shutdown", "N>=1", 3) * time.Second

	controlC := timeoutC.Config("control")
	conf.Timeout.Control = util.TimeoutConfig{
		Read:  controlC.DurationAndOr("read", "N>=3000", 10000) * time.Millisecond,
		Write: controlC.DurationAndOr("write", "N>0", 500) * time.Millisecond,
	}

	tunnelC := timeoutC.Config("tunnel")
	conf.Timeout.Tunnel = util.TimeoutConfig{
		Read:  tunnelC.DurationAndOr("read", "N>=100", 2000) * time.Millisecond,
		Write: tunnelC.DurationAndOr("write", "N>0", 300) * time.Millisecond,
	}

	return conf
}

func (conf *Config) BuildRPCServerConf(validateFunc birpc.ValidateFunc) *birpc.ServerConfig {
	rpcconf := &birpc.ServerConfig{}
	rpcconf.ListenAddr = conf.ControlAddr
	rpcconf.Timeout = conf.Timeout.Control
	rpcconf.TLSConf = &tls.Config{}
	rpcconf.ValidateFunc = validateFunc
	return rpcconf
}

func BuildWebConfig(rawConf cc.Configer) *web.Config {
	conf := &web.Config{}
	conf.Addr = rawConf.String("addr")
	conf.MuxDomain = rawConf.String("mux_domain")
	conf.HostIP = rawConf.String("host_ip") // proxy ip
	if conf.HostIP == "" {
		conf.HostIP, _ = util.HostIP()
	}
	conf.AllowOrigins = []string{}
	for _, origin := range rawConf.Value("allow_origins").List() {
		conf.AllowOrigins = append(conf.AllowOrigins, origin.String())
	}
	conf.MaxAdminAgents = rawConf.IntAndOr("max_admin_agents", "N>=10&&N=<23", 23)
	conf.MaxAdminTunnels = rawConf.IntAndOr("max_admin_tunnels", "N>=5&&N<=23", 23)
	conf.MaxUserAgents = rawConf.IntAndOr("max_user_agents", "N>=3&&N<=10", 5)
	conf.MaxUserTunnels = rawConf.IntAndOr("max_user_tunnels", "N>=3&&N<=12", 10)
	conf.MaxDownloadsPerHour = rawConf.IntAndOr("max_tunnel_updates_per_hour", "N>=5&&N<=24", 12)
	conf.MaxTunnelUpdatePerHour = rawConf.IntAndOr("max_downloads_per_hour", "N>=5&&N<=24", 8)
	conf.ClientConfig = rawConf.String("client_config")
	return conf
}
