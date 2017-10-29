package web

import (
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	"golang.org/x/crypto/bcrypt"

	"github.com/damnever/sunflower/sun/storage"
)

func (s *Server) login(c echo.Context) error {
	username := c.FormValue("username")
	password := c.FormValue("password")

	user, err := s.db.QueryUser(username)
	if err != nil {
		if storage.IsNotExist(err) {
			return newUserError("No such user")
		}
		return err
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err == bcrypt.ErrMismatchedHashAndPassword {
		return newUserError("wrong password")
	}
	if err != nil {
		return err
	}

	sess, _ := session.Get(SessionUser, c)
	sess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
	}
	sess.Values[SessionUserNameKey] = user.Name
	sess.Save(c.Request(), c.Response())

	return c.NoContent(http.StatusOK)
}

func (s *Server) logout(c echo.Context) error {
	sess, _ := session.Get(SessionUser, c)
	sess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	}
	sess.Save(c.Request(), c.Response())
	return c.NoContent(http.StatusOK)
}
