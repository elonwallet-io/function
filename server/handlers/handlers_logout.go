package handlers

import (
	"github.com/labstack/echo/v4"
	"net/http"
	"time"
)

func (a *Api) Logout() echo.HandlerFunc {
	return func(c echo.Context) error {
		c.SetCookie(&http.Cookie{
			Name:     "session",
			Value:    "",
			Expires:  time.Unix(0, 0),
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			Path:     "/",
		})
		return c.NoContent(http.StatusOK)
	}
}
