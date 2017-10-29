package web

import (
	"net/http"

	"github.com/labstack/echo"

	"github.com/damnever/sunflower/pkg/util"
	"github.com/damnever/sunflower/sun/pubsub"
	"github.com/damnever/sunflower/sun/storage"
)

func (s *Server) showAgents(c echo.Context) error {
	user := c.Get(CtxUser).(userCtx)
	agents, err := s.db.QueryAgents(user.targetName)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, agents)
}

func (s *Server) deleteAgents(c echo.Context) error {
	user := c.Get(CtxUser).(userCtx)
	if err := s.db.DeleteAgents(user.targetName); err != nil {
		return err
	}
	return c.NoContent(http.StatusOK)
}

func (s *Server) showAgent(c echo.Context) error {
	user := c.Get(CtxUser).(userCtx)
	ahash := c.Param("ahash")
	agent, err := s.db.QueryAgent(user.targetName, ahash)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, agent)
}

func (s *Server) createAgent(c echo.Context) error {
	user := c.Get(CtxUser).(userCtx)
	count, err := s.db.QueryAgentCount(user.targetName)
	if err != nil {
		return err
	}
	limits := s.conf.MaxUserAgents
	if user.isAdmin {
		limits = s.conf.MaxAdminAgents
	}
	if count > limits {
		return newUserError("only %d agents allowed", limits)
	}

	tag := c.FormValue("tag")
	if err := ValidateTag(tag); err != nil {
		return newUserError(err.Error())
	}

	ahash := util.Hash(user.targetName, tag)[:8]
	if err := s.db.CreateAgent(user.targetName, ahash, tag); err != nil {
		if storage.IsExist(err) {
			return newUserError("agent %s[%s] already exists, try again", ahash, tag)
		}
		return err
	}
	return c.JSON(http.StatusCreated, echo.Map{"hash": ahash})
}

func (s *Server) updateAgent(c echo.Context) error {
	return c.NoContent(http.StatusNotFound)
}

func (s *Server) deleteAgent(c echo.Context) error {
	user := c.Get(CtxUser).(userCtx)
	ahash := c.Param("ahash")
	if err := s.db.DeleteAgent(user.targetName, ahash); err != nil {
		return err
	}

	s.pub.Pub(ahash, &pubsub.Event{Type: pubsub.EventRejectAgent})
	return c.NoContent(http.StatusOK)
}
