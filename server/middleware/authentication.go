package middleware

import (
	"crypto/ed25519"
	"github.com/Leantar/elonwallet-function/server/common"
	"github.com/labstack/echo/v4"
	"golang.org/x/exp/slices"
	"net/http"
	"time"
)

const invalidSession = "Invalid or malformed jwt"

func CheckAuthentication(repo common.Repository, pk ed25519.PublicKey, additionalScopes ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user, err := repo.GetUser()
			if err != nil {
				return err
			}

			claims, err := checkJWT(c, user.Email, pk)
			if err != nil {
				return err
			}

			if !isAllowedScope(additionalScopes, claims) {
				return echo.NewHTTPError(http.StatusUnauthorized, invalidSession)
			}

			c.Set("claims", claims)
			c.Set("user", user)

			return next(c)
		}
	}
}

func CheckStrictAuthentication(repo common.Repository, pk ed25519.PublicKey, additionalScopes ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user, err := repo.GetUser()
			if err != nil {
				return err
			}

			claims, err := checkJWT(c, user.Email, pk)
			if err != nil {
				return err
			}

			iat, err := claims.GetIssuedAt()
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, invalidSession)
			}

			if time.Now().After(iat.Add(time.Minute * 15)) {
				return echo.NewHTTPError(http.StatusForbidden, "This session is too old to access this resource")
			}

			if !isAllowedScope(additionalScopes, claims) {
				return echo.NewHTTPError(http.StatusUnauthorized, invalidSession)
			}

			c.Set("claims", claims)
			c.Set("user", user)

			return next(c)
		}
	}
}

func checkJWT(c echo.Context, email string, pk ed25519.PublicKey) (common.EnclaveClaims, error) {
	cookie, err := c.Request().Cookie("session")
	if err != nil {
		return common.EnclaveClaims{}, echo.NewHTTPError(http.StatusUnauthorized, "Missing session cookie")
	}

	claims, err := common.ValidateEnclaveJWT(cookie.Value, email, pk)
	if err != nil {
		return common.EnclaveClaims{}, echo.NewHTTPError(http.StatusUnauthorized, invalidSession).SetInternal(err)
	}

	return claims, nil
}

func isAllowedScope(scopes []string, claims common.EnclaveClaims) bool {
	return claims.Scope == "all" || slices.Contains(scopes, claims.Scope)
}
