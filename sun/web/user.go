package web

import (
	"net/http"

	"github.com/labstack/echo"

	"github.com/damnever/sunflower/pkg/util"
	"github.com/damnever/sunflower/sun/pubsub"
	"github.com/damnever/sunflower/sun/storage"
)

func (s *Server) showUsers(c echo.Context) error {
	users, err := s.db.QueryUsers()
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, users)
}

func (s *Server) showUser(c echo.Context) error {
	curUser := c.Get(CtxUser).(userCtx)
	user, err := s.db.QueryUser(curUser.targetName)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, user)
}

func (s *Server) createUser(c echo.Context) error {
	// XXX(damnver): like shit...
	username := c.FormValue("username")
	if err := ValidateUsername(username); err != nil {
		return newUserError(err.Error())
	}
	password := c.FormValue("password")
	if err := ValidatePassword(password); err != nil {
		return newUserError(err.Error())
	}
	email := c.FormValue("email")
	if err := ValidateEmail(email); err != nil {
		return newUserError(err.Error())
	}
	password, err := util.EncryptPasswd([]byte(password))
	if err != nil {
		return err
	}

	err = s.db.CreateUser(username, password, email, false)
	if err != nil {
		if storage.IsExist(err) {
			return newUserError("user %s already exists", username)
		}
		return err
	}
	return c.NoContent(http.StatusCreated)
}

func (s *Server) updateUser(c echo.Context) error {
	fields := map[string]interface{}{}
	if password := c.FormValue("password"); password != "" {
		password, err := util.EncryptPasswd([]byte(password))
		if err != nil {
			return err
		}
		fields["password"] = password
	}
	if email := c.FormValue("email"); email != "" {
		if err := ValidateEmail(email); err != nil {
			return newUserError(err.Error())
		}
		fields["email"] = email
	}
	if len(fields) == 0 {
		return newUserError("empty fields")
	}

	user := c.Get(CtxUser).(userCtx)
	_, err := s.db.UpdateUser(user.targetName, fields)
	if err != nil {
		return err
	}
	return c.NoContent(http.StatusResetContent)
}

func (s *Server) deleteUser(c echo.Context) error {
	user := c.Get(CtxUser).(userCtx)
	ahashs, err := s.db.QueryAgentHashs(user.targetName)
	if err != nil && !storage.IsNotExist(err) {
		return err
	}
	if err := s.db.DeleteUser(user.targetName); err != nil {
		return err
	}

	for _, ahash := range ahashs {
		s.pub.Pub(ahash, &pubsub.Event{Type: pubsub.EventRejectAgent})
	}
	return c.NoContent(http.StatusOK)
}

// place it here for now..
func (s *Server) showAgentsAndTunnels(c echo.Context) error {
	user := c.Get(CtxUser).(userCtx)
	agents, err := s.db.QueryAgents(user.targetName)
	if err != nil {
		return err
	}

	for i := range agents {
		agent := &(agents[i])
		tunnels, err := s.db.QueryTunnels(user.targetName, agent.Hash)
		if err != nil {
			return err
		}
		agent.Tunnels = tunnels
	}
	return c.JSON(http.StatusOK, agents)
}
