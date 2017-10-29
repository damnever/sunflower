package web

import (
	"net/http"

	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
)

func (s *Server) authChecker(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		sess, _ := session.Get(SessionUser, c)
		username, ok := sess.Values[SessionUserNameKey]
		if !ok {
			return c.NoContent(http.StatusUnauthorized)
		}
		user, err := s.db.QueryUser(username.(string))
		if err != nil {
			return c.NoContent(http.StatusUnauthorized)
		}

		targetUsername := c.Param("username")
		if targetUsername == "" {
			targetUsername = user.Name
		}

		c.Set(CtxUser, userCtx{
			name:       user.Name,
			targetName: targetUsername,
			isAdmin:    user.IsAdmin,
		})
		return next(c)
	}
}

func (s *Server) adminChecker(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		user := c.Get(CtxUser).(userCtx)
		if !user.isAdmin {
			return c.NoContent(http.StatusForbidden)
		}
		return next(c)
	}
}
