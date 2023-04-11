package middleware

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"net/http"
)

func Cors(frontendURL string) echo.MiddlewareFunc {
	return middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{frontendURL},
		AllowMethods:     []string{http.MethodHead, http.MethodGet, http.MethodPost, http.MethodDelete, http.MethodPut},
		AllowCredentials: true,
	})
}
