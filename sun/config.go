package sun

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/damnever/cc"

	"github.com/damnever/sunflower/birpc"
	"github.com/damnever/sunflower/sun/registry"
	"github.com/damnever/sunflower/sun/web"
)

type Config struct {
	GracefulShutdown time.Duration
	MuxRegConf       registry.Config
	RPCConf          birpc.ServerConfig
}

func buildCoreConfig(rawConf cc.Configer) Config {
	conf := Config{}
	conf.GracefulShutdown = rawConf.DurationAndOr("graceful_shutdown", "N>=1", 3) * time.Second
	{
		rpcconf := birpc.ServerConfig{}
		controlC := rawConf.Config("control")
		rpcconf.ListenAddr = controlC.String("addr")
		timeoutC := controlC.Config("timeout")
		rpcconf.Timeout.Read = timeoutC.DurationAndOr("read", "N>=3000", 10000) * time.Millisecond
		rpcconf.Timeout.Write = timeoutC.DurationAndOr("write", "N>0", 500) * time.Millisecond
		rpcconf.TLSConf = &tls.Config{}
		conf.RPCConf = rpcconf
	}
	{
		mrconf := registry.Config{}
		mrconf.Domain = rawConf.String("domain")
		mrconf.IP = rawConf.StringOr("proxy_ip", rawConf.String("host_ip"))
		muxC := rawConf.Config("muxreg")
		mrconf.HTTPAddr = muxC.String("http_addr")
		timeoutC := muxC.Config("timeout")
		mrconf.Timeout.Read = timeoutC.DurationAndOr("read", "N>=100", 2000) * time.Millisecond
		mrconf.Timeout.Write = timeoutC.DurationAndOr("write", "N>0", 300) * time.Millisecond
		conf.MuxRegConf = mrconf
	}
	return conf
}

func buildWebConfig(controlPort string, rawConf cc.Configer) *web.Config {
	conf := &web.Config{}
	conf.HostIP = rawConf.StringOr("proxy_ip", rawConf.String("host_ip"))
	conf.MuxDomain = rawConf.String("domain")
	webC := rawConf.Config("web")
	conf.Addr = webC.String("addr")
	conf.AllowOrigins = []string{}
	for _, origin := range webC.Value("allow_origins").List() {
		conf.AllowOrigins = append(conf.AllowOrigins, origin.String())
	}
	conf.MaxAdminAgents = webC.IntAndOr("max_admin_agents", "N>=10&&N=<23", 23)
	conf.MaxAdminTunnels = webC.IntAndOr("max_admin_tunnels", "N>=5&&N<=23", 23)
	conf.MaxUserAgents = webC.IntAndOr("max_user_agents", "N>=3&&N<=10", 5)
	conf.MaxUserTunnels = webC.IntAndOr("max_user_tunnels", "N>=3&&N<=12", 10)
	conf.MaxDownloadsPerHour = webC.IntAndOr("max_tunnel_updates_per_hour", "N>=5&&N<=24", 12)
	conf.MaxTunnelUpdatePerHour = webC.IntAndOr("max_downloads_per_hour", "N>=5&&N<=24", 8)
	agentConfig := fmt.Sprintf("control_server: %s:%s\n%s", conf.HostIP, controlPort, webC.String("agent_config"))
	conf.AgentConfig = agentConfig
	return conf
}
