package middleware

import (
	"crypto/ed25519"
	"github.com/Leantar/elonwallet-function/server/common"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"net/http"
	"time"
)

func CheckAuthentication(pk ed25519.PublicKey) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims, err := checkJWT(c, pk)
			if err != nil {
				return err
			}

			c.Set("claims", claims)

			return next(c)
		}
	}
}

func CheckStrictAuthentication(pk ed25519.PublicKey) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims, err := checkJWT(c, pk)
			if err != nil {
				return err
			}

			iat, err := claims.GetIssuedAt()
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid session cookie")
			}

			// Check if the session is older than 15 minutes
			if iat.Add(time.Minute * 15).Before(time.Now()) {
				return echo.NewHTTPError(http.StatusForbidden, "This session is too old to access this resource")
			}

			c.Set("claims", claims)

			return next(c)
		}
	}
}

func checkJWT(c echo.Context, pk ed25519.PublicKey) (jwt.MapClaims, error) {
	cookie, err := c.Request().Cookie("session")
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "Missing session cookie")
	}

	claims, ok := common.ValidateJWT(cookie.Value, pk)
	if !ok {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "Invalid session cookie")
	}

	return claims, nil
}
