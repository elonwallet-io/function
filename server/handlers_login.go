package server

import (
	"bytes"
	"fmt"
	"github.com/Leantar/elonwallet-function/models"
	"github.com/labstack/echo/v4"
	"net/http"
	"time"
)

func (a *Api) LoginInitialize() echo.HandlerFunc {
	return func(c echo.Context) error {
		user := c.Get("user").(*models.User)

		options, session, err := a.w.BeginLogin(user.WebauthnData)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		user.WebauthnData.Sessions[LoginKey] = *session

		return c.JSON(http.StatusOK, options)
	}
}

// LoginFinalize proxies the login of a user
func (a *Api) LoginFinalize() echo.HandlerFunc {
	return func(c echo.Context) error {
		user := c.Get("user").(*models.User)
		session, ok := user.WebauthnData.Sessions[LoginKey]
		if !ok {
			return echo.NewHTTPError(http.StatusBadRequest, "Login must be initialized beforehand")
		}
		delete(user.WebauthnData.Sessions, LoginKey)

		cred, err := a.w.FinishLogin(user.WebauthnData, session, c.Request())
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		var credName string
		for name, credential := range user.WebauthnData.Credentials {
			if bytes.Equal(credential.ID, cred.ID) {
				credName = name
				break
			}
		}

		jwtString, err := CreateJWT(credName, user.JWTSigningKey)
		if err != nil {
			return fmt.Errorf("failed to create jwt: %w", err)
		}

		c.SetCookie(&http.Cookie{
			Name:     "session",
			Value:    jwtString,
			Expires:  time.Now().Add(time.Hour * 24),
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			Path:     "/",
		})

		return c.NoContent(http.StatusOK)
	}
}
