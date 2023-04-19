package middleware

import (
	"crypto/ed25519"
	"fmt"
	"github.com/Leantar/elonwallet-function/server/common"
	"github.com/labstack/echo/v4"
	"net/http"
	"time"
)

func CheckAuthentication(repo common.Repository, pk ed25519.PublicKey, additionalScopes ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user, err := repo.GetUser()
			if err != nil {
				return fmt.Errorf("failed to get user: %w", err)
			}

			claims, err := checkJWT(c, user.Email, pk)
			if err != nil {
				return err
			}

			if len(additionalScopes) > 0 && !checkScopes(additionalScopes, claims) {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid or malformed session")
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
				return fmt.Errorf("failed to get user: %w", err)
			}

			claims, err := checkJWT(c, user.Email, pk)
			if err != nil {
				return err
			}

			iat, err := claims.GetIssuedAt()
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid session cookie")
			}

			if time.Now().After(iat.Add(time.Minute * 15)) {
				return echo.NewHTTPError(http.StatusForbidden, "This session is too old to access this resource")
			}

			if len(additionalScopes) > 0 && !checkScopes(additionalScopes, claims) {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid or malformed session")
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
		return common.EnclaveClaims{}, echo.NewHTTPError(http.StatusUnauthorized, "Invalid or malformed session").SetInternal(err)
	}

	return claims, nil
}

func checkScopes(scopes []string, claims common.EnclaveClaims) bool {
	for _, scope := range scopes {
		if scope == "all" || claims.Scope == scope {
			return true
		}
	}

	return false
}
