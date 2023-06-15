package middleware

import (
	"crypto/ed25519"
	"fmt"
	"github.com/Leantar/elonwallet-function/config"
	"github.com/Leantar/elonwallet-function/models"
	"github.com/Leantar/elonwallet-function/server/common"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"golang.org/x/exp/slices"
	"net/http"
	"time"
)

const (
	invalidSession = "Invalid or malformed jwt"
)

func CheckAuthentication(repo common.Repository, pk ed25519.PublicKey, allowedScopes ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user, claims, err := frontendAuth(c, repo, pk, allowedScopes)
			if err != nil {
				return err
			}

			c.Set("claims", claims)
			c.Set("user", user)

			return next(c)
		}
	}
}

func CheckStrictAuthentication(repo common.Repository, pk ed25519.PublicKey, allowedScopes ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user, claims, err := frontendAuth(c, repo, pk, allowedScopes)
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

			c.Set("claims", claims)
			c.Set("user", user)

			return next(c)
		}
	}
}

func CheckEnclaveAuthentication(repo common.Repository, cfg config.Config) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			bearer := c.Request().Header.Get("Authorization")
			if len(bearer) < 8 {
				return echo.NewHTTPError(http.StatusUnauthorized, invalidSession)
			}

			user, err := repo.GetUser()
			if err != nil {
				return err
			}

			claims, err := common.ValidateJWT(bearer[7:], enclaveKeyFunc(cfg, user, c))
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, invalidSession).SetInternal(err)
			}

			if claims.Scope != "enclave" {
				return echo.NewHTTPError(http.StatusUnauthorized, invalidSession)
			}

			c.Set("claims", claims)
			c.Set("user", user)

			return next(c)
		}
	}
}

func frontendAuth(c echo.Context, repo common.Repository, pk ed25519.PublicKey, allowedScopes []string) (models.User, common.EnclaveClaims, error) {
	cookie, err := c.Request().Cookie("session")
	if err != nil {
		return models.User{}, common.EnclaveClaims{}, echo.NewHTTPError(http.StatusUnauthorized, invalidSession)
	}

	user, err := repo.GetUser()
	if err != nil {
		return models.User{}, common.EnclaveClaims{}, err
	}

	claims, err := common.ValidateJWT(cookie.Value, frontendKeyFunc(pk))
	if err != nil {
		return models.User{}, common.EnclaveClaims{}, echo.NewHTTPError(http.StatusUnauthorized, invalidSession).SetInternal(err)
	}

	if !isAllowedScope(allowedScopes, claims) {
		return models.User{}, common.EnclaveClaims{}, echo.NewHTTPError(http.StatusUnauthorized, invalidSession)
	}

	if claims.Subject != user.Email {
		return models.User{}, common.EnclaveClaims{}, echo.NewHTTPError(http.StatusUnauthorized, invalidSession)
	}

	return user, claims, nil
}

func isAllowedScope(scopes []string, claims common.EnclaveClaims) bool {
	return slices.Contains(scopes, claims.Scope)
}

func frontendKeyFunc(pk ed25519.PublicKey) jwt.Keyfunc {
	return func(token *jwt.Token) (interface{}, error) {
		return pk, nil
	}
}

func enclaveKeyFunc(cfg config.Config, user models.User, c echo.Context) jwt.Keyfunc {
	return func(token *jwt.Token) (interface{}, error) {
		email, err := token.Claims.GetSubject()
		if err != nil {
			return nil, err
		}

		enclaveURL, err := getEnclaveURL(cfg, user, email)
		if err != nil {
			return nil, err
		}

		enclaveApiClient, err := common.NewEnclaveApiClient(enclaveURL, models.User{}, nil)
		if err != nil {
			return nil, err
		}

		c.Set("enclave_url", enclaveURL)

		jwtSigningKey, err := enclaveApiClient.GetJWTVerificationKey()
		if err != nil {
			return nil, err
		}

		return jwtSigningKey, nil
	}
}

func getEnclaveURL(cfg config.Config, user models.User, email string) (string, error) {
	if grant, ok := user.EmergencyAccessGrants[email]; ok {
		return grant.EnclaveURL, nil
	} else {
		backendApiClient, err := common.NewBackendApiClient(cfg.BackendURL, models.User{}, nil)
		if err != nil {
			return "", fmt.Errorf("failed to create backend api client: %w", err)
		}

		enclaveURL, err := backendApiClient.GetEnclaveURL(email)
		if err != nil {
			return "", err
		}

		return enclaveURL, nil
	}
}
