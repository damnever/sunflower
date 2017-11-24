package web

import (
	"expvar"
	"fmt"
	"mime"
	"net/http"
	"net/http/pprof"
	"path/filepath"
	"strings"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/middleware"

	"github.com/damnever/sunflower/pkg/util"
	"github.com/damnever/sunflower/sun/pubsub"
	"github.com/damnever/sunflower/sun/storage"
)

const (
	SessionUser        string = "session-you-guess"
	SessionUserNameKey        = "session-username"
	CtxUser                   = "ctx-user"
)

type userCtx struct {
	name       string
	targetName string
	isAdmin    bool
}

type Config struct {
	Addr                   string
	DataDir                string
	MuxDomain              string
	AllowOrigins           []string
	HostIP                 string
	AgentConfig            string
	MaxAdminAgents         int
	MaxAdminTunnels        int
	MaxUserAgents          int
	MaxUserTunnels         int
	MaxDownloadsPerHour    int
	MaxTunnelUpdatePerHour int
}

type Server struct {
	e                *echo.Echo
	conf             *Config
	downloadsCounter *counter
	updatesCounter   *counter
	builder          *Builder
	db               *storage.DB
	pub              pubsub.Publisher
}

func New(conf *Config, db *storage.DB, pub pubsub.Publisher) (*Server, error) {
	builder, err := NewBuilder(conf.DataDir, conf.AgentConfig)
	if err != nil {
		return nil, err
	}

	s := &Server{
		e:                echo.New(),
		conf:             conf,
		downloadsCounter: newCounster(conf.MaxDownloadsPerHour),
		updatesCounter:   newCounster(conf.MaxTunnelUpdatePerHour),
		builder:          builder,
		db:               db,
		pub:              pub,
	}
	s.e.HideBanner = true
	s.setupMiddlewares()
	s.setupRouters()
	return s, nil
}

func (s *Server) Serve() error {
	go s.builder.StartCrossPlatformBuild()
	return s.e.Start(s.conf.Addr)
}

func (s *Server) Endpoint() string {
	addr := s.conf.Addr
	if strings.HasPrefix(addr, ":") {
		addr = "0.0.0.0" + addr
	}
	return fmt.Sprintf("http://%s", addr)
}

func (s *Server) Close() error {
	s.builder.Cancel()
	return s.e.Close()
}

func (s *Server) setupMiddlewares() {
	e := s.e

	e.Pre(middleware.RemoveTrailingSlash())

	e.Use(middleware.Logger()) // Custom logger
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: s.conf.AllowOrigins,
	}))
	e.Use(session.MiddlewareWithConfig(session.Config{
		Store: sessions.NewCookieStore([]byte(util.RandString(14))),
	}))
}

func (s *Server) setupRouters() {
	s.registerConfigAPIRouter()
	s.registerAssetsRouter()
	s.registerAuthAPIRouters()
	s.registerUserAPIRouters()
	s.registerAdminAPIRouters()
	s.registerSysAPIRouters()
}

func (s *Server) registerAssetsRouter() {
	for _, name := range AssetNames() {
		if name == "flower.zip" {
			continue
		}
		var pattern string
		if name == "index.html" {
			pattern = "/"
		} else {
			pattern = fmt.Sprintf("/%s", name)
		}

		s.e.GET(pattern, func(name string) echo.HandlerFunc {
			return func(c echo.Context) error {
				mimeType := mime.TypeByExtension(filepath.Ext(name))
				return c.Blob(http.StatusOK, mimeType, MustAsset(name))
			}
		}(name)) // Remember the name..
	}
}

func (s *Server) registerConfigAPIRouter() {
	s.e.GET("/api/config", func(c echo.Context) error {
		return c.JSON(http.StatusOK, echo.Map{
			"domain": s.conf.MuxDomain,
			"ip":     s.conf.HostIP,
		})
	})
}

func (s *Server) registerAuthAPIRouters() {
	e := s.e
	e.POST("/api/login", s.login)
	e.DELETE("/api/logout", s.logout)
}

func (s *Server) registerUserAPIRouters() {
	g := s.e.Group("/api/user")
	g.Use(s.authChecker)

	g.GET("", s.showUser)
	g.PATCH("", s.updateUser)
	g.PUT("", s.updateUser)
	g.DELETE("", s.deleteUser)

	g.GET("/agents", s.showAgents)
	g.DELETE("/agents", s.deleteAgents)
	g.POST("/agents", s.createAgent)
	g.GET("/agents/:ahash", s.showAgent)
	g.PATCH("/agents/:ahash", s.updateAgent)
	g.PUT("/agents/:ahash", s.updateAgent)
	g.DELETE("/agents/:ahash", s.deleteAgent)

	g.GET("/agents/:ahash/bin", s.download)

	g.GET("/agents/:ahash/tunnels", s.showTunnels)
	g.DELETE("/agents/:ahash/tunnels", s.deleteTunnels)
	g.POST("/agents/:ahash/tunnels", s.createTunnel)
	g.GET("/agents/:ahash/tunnels/:thash", s.showTunnel)
	g.PATCH("/agents/:ahash/tunnels/:thash", s.updateTunnel)
	g.PUT("/agents/:ahash/tunnels/:thash", s.updateTunnel)
	g.DELETE("/agents/:ahash/tunnels/:thash", s.deleteTunnel)
}

func (s *Server) registerAdminAPIRouters() {
	g := s.e.Group("/api/users")
	g.Use(s.authChecker)
	g.Use(s.adminChecker)

	g.GET("", s.showUsers)
	g.POST("", s.createUser)
	g.GET("/:username", s.showUser)
	g.PATCH("/:username", s.updateUser)
	g.PUT("/:username", s.updateUser)
	g.DELETE("/:username", s.deleteUser)

	g.GET("/:username/agents", s.showAgents)
	g.DELETE("/:username/agents", s.deleteAgents)
	g.POST("/:username/agents", s.createAgent)
	g.GET("/:username/agents/:ahash", s.showAgent)
	g.DELETE("/:username/agents/:ahash", s.deleteAgent)

	g.GET("/:username/agents/:ahash/bin", s.download)

	g.GET("/:username/agents/:ahash/tunnels", s.showTunnels)
	g.DELETE("/:username/agents/:ahash/tunnels", s.deleteTunnels)
	g.POST("/:username/agents/:ahash/tunnels", s.createTunnel)
	g.GET("/:username/agents/:ahash/tunnels/:thash", s.showTunnel)
	g.PATCH("/:username/agents/:ahash/tunnels/:thash", s.updateTunnel)
	g.PUT("/:username/agents/:ahash/tunnels/:thash", s.updateTunnel)
	g.DELETE("/:username/agents/:ahash/tunnels/:thash", s.deleteTunnel)
}

func (s *Server) registerSysAPIRouters() {
	g := s.e.Group("/api/sys")
	g.Use(s.authChecker)
	g.Use(s.adminChecker)

	g.GET("/stats", s.stats)

	for _, name := range []string{
		"goroutine",
		"heap",
		"block",
		"threadcreate",
	} {
		g.GET("/debug/pprof/"+name, echo.WrapHandler(pprof.Handler(name)))
	}
	g.Any("/debug/pprof/profile", func(c echo.Context) error {
		pprof.Profile(c.Response().Writer, c.Request())
		return nil
	})
	g.Any("/debug/pprof/symbol", func(c echo.Context) error {
		pprof.Symbol(c.Response().Writer, c.Request())
		return nil
	})
	g.Any("/debug/pprof/trace", func(c echo.Context) error {
		pprof.Trace(c.Response().Writer, c.Request())
		return nil
	})
	g.GET("/debug/vars", echo.WrapHandler(expvar.Handler()))
}

func newAuthError(format string, args ...interface{}) *echo.HTTPError {
	msg := fmt.Sprintf(format, args...)
	return &echo.HTTPError{
		Code:    http.StatusUnauthorized,
		Message: msg,
	}
}

func newUserError(format string, args ...interface{}) *echo.HTTPError {
	msg := fmt.Sprintf(format, args...)
	return &echo.HTTPError{
		Code:    http.StatusBadRequest,
		Message: msg,
	}
}
