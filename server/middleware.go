package server

import (
	"crypto/ed25519"
	"fmt"
	"github.com/labstack/echo/v4"
	"net/http"
)

func (a *Api) checkAuthentication() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cookie, err := c.Request().Cookie("session")
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "Missing session cookie")
			}

			user, err := a.d.GetUser()
			if err != nil {
				return fmt.Errorf("failed to get user: %w", err)
			}

			pk := user.JWTSigningKey.Public().(ed25519.PublicKey)

			claims, ok := ValidateJWT(cookie.Value, pk)
			if !ok {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid session cookie")
			}

			c.Set("claims", claims)

			return next(c)
		}
	}
}

func (a *Api) manageUser() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Must lock user from datastore because it is not thread safe
			a.mu.Lock()
			defer a.mu.Unlock()

			user, err := a.d.GetUser()
			if err != nil {
				return fmt.Errorf("failed to get user: %w", err)
			}
			c.Set("user", &user)

			err = next(c)
			if err != nil {
				return err
			}

			err = a.d.SaveUser(user)
			if err != nil {
				return fmt.Errorf("failed to save user: %w", err)
			}

			return nil
		}
	}
}
