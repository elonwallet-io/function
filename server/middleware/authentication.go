package middleware

import (
	"crypto/ed25519"
	"errors"
	"github.com/Leantar/elonwallet-function/models"
	"github.com/Leantar/elonwallet-function/server/common"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"golang.org/x/exp/slices"
	"net/http"
	"time"
)

const invalidSession = "Invalid or malformed jwt"

func CheckAuthentication(repo common.Repository, pk ed25519.PublicKey, additionalScopes ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user, claims, err := checkDefaultAuth(c, repo, pk, additionalScopes...)
			if err != nil {
				return err
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
			user, claims, err := checkDefaultAuth(c, repo, pk, additionalScopes...)
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

func CheckEmergencyContactAuthentication(repo common.Repository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			bearer := c.Request().Header.Get("Authorization")
			if len(bearer) < 8 {
				return echo.NewHTTPError(http.StatusUnauthorized, "Missing or invalid session")
			}

			user, err := repo.GetUser()
			if err != nil {
				return err
			}

			claims, err := common.ValidateJWT(bearer[7:], trustedEmergencyContactKeyFunc(user))
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, invalidSession).SetInternal(err)
			}

			contact := user.EmergencyAccessContacts[claims.Subject]

			c.Set("contact", contact)
			c.Set("user", user)

			return next(c)
		}
	}
}

func checkDefaultAuth(c echo.Context, repo common.Repository, pk ed25519.PublicKey, additionalScopes ...string) (models.User, common.EnclaveClaims, error) {
	cookie, err := c.Request().Cookie("session")
	if err != nil {
		return models.User{}, common.EnclaveClaims{}, echo.NewHTTPError(http.StatusUnauthorized, "Missing session cookie")
	}

	user, err := repo.GetUser()
	if err != nil {
		return models.User{}, common.EnclaveClaims{}, err
	}

	claims, err := common.ValidateJWT(cookie.Value, defaultKeyFunc(pk))
	if err != nil {
		return models.User{}, common.EnclaveClaims{}, echo.NewHTTPError(http.StatusUnauthorized, invalidSession).SetInternal(err)
	}

	if !isAllowedScope(additionalScopes, claims) {
		return models.User{}, common.EnclaveClaims{}, echo.NewHTTPError(http.StatusUnauthorized, invalidSession)
	}

	if claims.Subject != user.Email {
		return models.User{}, common.EnclaveClaims{}, echo.NewHTTPError(http.StatusUnauthorized, invalidSession)
	}

	return user, claims, nil
}

func isAllowedScope(scopes []string, claims common.EnclaveClaims) bool {
	return claims.Scope == "all" || slices.Contains(scopes, claims.Scope)
}

func defaultKeyFunc(pk ed25519.PublicKey) jwt.Keyfunc {
	return func(token *jwt.Token) (interface{}, error) {
		return pk, nil
	}
}

func trustedEmergencyContactKeyFunc(user models.User) jwt.Keyfunc {
	return func(token *jwt.Token) (interface{}, error) {
		email, err := token.Claims.GetSubject()
		if err != nil {
			return nil, err
		}

		contact, ok := user.EmergencyAccessContacts[email]
		if !ok {
			return nil, errors.New("emergency contact not found")
		}

		return contact.JWTSigningKey, nil
	}
}
