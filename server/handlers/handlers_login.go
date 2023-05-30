package handlers

import (
	"bytes"
	"crypto/ed25519"
	"fmt"
	"github.com/Leantar/elonwallet-function/models"
	"github.com/Leantar/elonwallet-function/server/common"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/labstack/echo/v4"
	"net/http"
	"time"
)

func (a *Api) HandleLoginInitialize() echo.HandlerFunc {
	return func(c echo.Context) error {
		user, err := a.repo.GetUser()
		if err != nil {
			return err
		}

		options, err := a.loginInitialize(&user, LoginKey)
		if err != nil {
			return err
		}

		err = a.repo.UpsertUser(user)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, options)
	}
}

func (a *Api) HandleLoginFinalize() echo.HandlerFunc {
	type output struct {
		BackendJWT string `json:"backend_jwt"`
	}
	return func(c echo.Context) error {
		user, err := a.repo.GetUser()
		if err != nil {
			return err
		}

		cred, _, err := a.loginFinalize(&user, c.Request(), LoginKey)
		if err != nil {
			return err
		}

		err = a.repo.UpsertUser(user)
		if err != nil {
			return err
		}

		cookie, err := createSessionCookie(user, cred, a.signingKey.PrivateKey)
		if err != nil {
			return err
		}

		c.SetCookie(cookie)

		//create an auth token to be used with the backend
		jwtString, err := common.CreateBackendJWT(user, common.ScopeUser, a.signingKey.PrivateKey)
		if err != nil {
			return fmt.Errorf("failed to create jwt: %w", err)
		}

		return c.JSON(http.StatusOK, output{jwtString})
	}
}

func createSessionCookie(user models.User, currentCredential *webauthn.Credential, sk ed25519.PrivateKey) (*http.Cookie, error) {
	var currentCredentialName string
	for name, credential := range user.WebauthnData.Credentials {
		if bytes.Equal(credential.ID, currentCredential.ID) {
			currentCredentialName = name
			break
		}
	}

	jwt, err := common.CreateEnclaveJWT(user, common.ScopeUser, currentCredentialName, sk)
	if err != nil {
		return nil, fmt.Errorf("failed to create jwt: %w", err)
	}

	return &http.Cookie{
		Name:     "session",
		Value:    jwt,
		Expires:  time.Now().Add(time.Hour * 24),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	}, nil
}
