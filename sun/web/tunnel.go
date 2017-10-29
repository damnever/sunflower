package web

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo"

	"github.com/damnever/sunflower/pkg/util"
	"github.com/damnever/sunflower/sun/pubsub"
	"github.com/damnever/sunflower/sun/storage"
)

func (s *Server) showTunnels(c echo.Context) error {
	user := c.Get(CtxUser).(userCtx)
	ahash := c.Param("ahash")
	tunnels, err := s.db.QueryTunnels(user.targetName, ahash)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, tunnels)
}

func (s *Server) deleteTunnels(c echo.Context) error {
	user := c.Get(CtxUser).(userCtx)
	ahash := c.Param("ahash")
	if err := s.db.DeleteTunnels(user.targetName, ahash); err != nil {
		return err
	}
	return c.NoContent(http.StatusOK)
}

func (s *Server) showTunnel(c echo.Context) error {
	user := c.Get(CtxUser).(userCtx)
	ahash := c.Param("ahash")
	thash := c.Param("thash")
	tunnel, err := s.db.QueryTunnel(user.targetName, ahash, thash)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, tunnel)
}

func (s *Server) createTunnel(c echo.Context) error {
	// XXX(damnever): like shit....
	user := c.Get(CtxUser).(userCtx)
	ahash := c.Param("ahash")
	count, err := s.db.QueryTunnelCount(user.targetName, ahash)
	if err != nil {
		return err
	}
	limits := s.conf.MaxUserTunnels
	if user.isAdmin {
		limits = s.conf.MaxAdminTunnels
	}
	if count > limits {
		return newUserError("only %d tunnels allowed", limits)
	}

	proto := strings.ToUpper(c.FormValue("proto"))
	if err := ValidteProtocol(proto); err != nil {
		return newUserError(err.Error())
	}
	exportAddr := c.FormValue("export_addr")
	if err := ValidateAddr(exportAddr); err != nil {
		return newUserError(err.Error())
	}
	tag := c.FormValue("tag")
	if err := ValidateTag(tag); err != nil {
		return newUserError(err.Error())
	}

	serverAddr := c.FormValue("server_addr")
	if proto == "HTTP" {
		serverAddr = fmt.Sprintf("%s.%s", serverAddr, user.targetName)
	} else {
		serverAddr = fmt.Sprintf("0.0.0.0:%s", serverAddr)
		if err := ValidateAddr(serverAddr); err != nil {
			return newUserError(err.Error())
		}
	}
	thash := util.Hash(user.targetName, ahash, tag)[:8]
	err = s.db.CreateTunnel(user.targetName, ahash, thash, proto, exportAddr, serverAddr, tag)
	if err != nil {
		if storage.IsExist(err) {
			return newUserError("tunnel %s[%s] already exists, try again", thash, tag)
		}
		return err
	}

	s.pub.Pub(ahash, &pubsub.Event{
		Type:       pubsub.EventOpenTunnel,
		TunnelHash: thash,
	})
	return c.JSON(http.StatusCreated, echo.Map{"hash": thash})
}

func (s *Server) updateTunnel(c echo.Context) error {
	// TODO(damnever): update other fields
	user := c.Get(CtxUser).(userCtx)
	ahash := c.Param("ahash")
	thash := c.Param("thash")
	if !user.isAdmin && !s.updatesCounter.Incr(user.name, ahash, thash) {
		return newUserError("Exceed the limit of max tunnel updates per hour")
	}

	enabled := c.FormValue("enabled")
	if enabled == "" {
		return newUserError("empty fields")
	}
	params := map[string]interface{}{"enabled": true}
	evtType := pubsub.EventOpenTunnel
	if enabled == "false" {
		params["enabled"] = false
		evtType = pubsub.EventCloseTunnel
	}

	if _, err := s.db.UpdateTunnel(user.targetName, ahash, thash, params); err != nil {
		return err
	}

	s.pub.Pub(ahash, &pubsub.Event{
		Type:       evtType,
		TunnelHash: thash,
	})
	return c.NoContent(http.StatusResetContent)
}

func (s *Server) deleteTunnel(c echo.Context) error {
	user := c.Get(CtxUser).(userCtx)
	ahash := c.Param("ahash")
	thash := c.Param("thash")
	if err := s.db.DeleteTunnel(user.targetName, ahash, thash); err != nil {
		return err
	}

	s.pub.Pub(ahash, &pubsub.Event{
		Type:       pubsub.EventCloseTunnel,
		TunnelHash: thash,
	})
	return c.NoContent(http.StatusOK)
}
