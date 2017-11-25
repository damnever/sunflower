package web

import (
	"fmt"
	"mime"
	"net/http"

	"github.com/damnever/sunflower/version"
	"github.com/labstack/echo"
)

func (s *Server) download(c echo.Context) error {
	user := c.Get(CtxUser).(userCtx)
	ahash := c.Param("ahash")
	if !user.isAdmin && !s.downloadsCounter.Incr(user.name, ahash) {
		return newUserError("Exceed the limit of max agent downloads per hour")
	}

	os := c.QueryParam("GOOS")
	arch := c.QueryParam("GOARCH")
	arm := c.QueryParam("GOARM")
	configType := c.QueryParam("config_type")

	data, err := s.builder.TryGetPkg(user.targetName, ahash, os, arch, arm, configType)
	if err != nil {
		if err == errIsBuilding || err == errUnknown {
			return newUserError(err.Error())
		}
		return err
	}
	filename := fmt.Sprintf("attachment; filename=flower-%s-%s_%s%s.zip", version.Info(), os, arch, arm)
	c.Response().Header().Set("Content-Disposition", filename)
	return c.Blob(http.StatusOK, mime.TypeByExtension(".zip"), data)
}
