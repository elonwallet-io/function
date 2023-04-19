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

func (a *Api) LoginInitialize() echo.HandlerFunc {
	return func(c echo.Context) error {
		user, err := a.repo.GetUser()
		if err != nil {
			return fmt.Errorf("failed to get user: %w", err)
		}

		options, session, err := a.w.BeginLogin(user.WebauthnData)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		user.WebauthnData.Sessions[LoginKey] = *session
		err = a.repo.UpsertUser(user)
		if err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}

		return c.JSON(http.StatusOK, options)
	}
}

func (a *Api) LoginFinalize() echo.HandlerFunc {
	type output struct {
		BackendJWT string `json:"backend_jwt"`
	}
	return func(c echo.Context) error {
		user, err := a.repo.GetUser()
		if err != nil {
			return fmt.Errorf("failed to get user: %w", err)
		}

		session, ok := user.WebauthnData.Sessions[LoginKey]
		if !ok {
			return echo.NewHTTPError(http.StatusBadRequest, "Login must be initialized beforehand")
		}
		delete(user.WebauthnData.Sessions, LoginKey)

		cred, err := a.w.FinishLogin(user.WebauthnData, session, c.Request())
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		cookie, err := createSessionCookie(user, cred, a.signingKey.PrivateKey)
		if err != nil {
			return err
		}

		c.SetCookie(cookie)

		err = a.repo.UpsertUser(user)
		if err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}

		//create an auth token to be used with the backend
		jwtString, err := common.CreateBackendJWT(user, a.signingKey.PrivateKey)
		if err != nil {
			return fmt.Errorf("failed to create jwt: %w", err)
		}

		return c.JSON(http.StatusOK, output{jwtString})
	}
}

func (a *Api) LoginWithOneTimeCode() echo.HandlerFunc {
	type input struct {
		OTP string `json:"otp" validate:"is-otp"`
	}
	return func(c echo.Context) error {
		var in input
		if err := c.Bind(&in); err != nil {
			return err
		}
		if err := c.Validate(&in); err != nil {
			return err
		}

		user := c.Get("user").(models.User)

		if user.OTP.Secret != in.OTP {
			user.OTP.TimesTried++
			err := a.repo.UpsertUser(user)
			if err != nil {
				return fmt.Errorf("failed to upsert user: %w", err)
			}
			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid or nonexistent OTP")
		}

		if user.OTP.TimesTried > 2 {
			return echo.NewHTTPError(http.StatusUnauthorized, "Too many retries. Please request a new OTP")
		}

		if time.Now().After(time.Unix(user.OTP.ValidUntil, 0)) {
			return echo.NewHTTPError(http.StatusUnauthorized, "OTP has expired. Please request a new OTP")
		}

		cookie, err := createOTPSessionCookie(user, a.signingKey.PrivateKey)
		if err != nil {
			return err
		}

		c.SetCookie(cookie)

		return c.NoContent(http.StatusOK)
	}
}

func createOTPSessionCookie(user models.User, sk ed25519.PrivateKey) (*http.Cookie, error) {
	jwtString, err := common.CreateCredentialEnclaveJWT(user, sk)
	if err != nil {
		return nil, fmt.Errorf("failed to create jwt: %w", err)
	}

	return &http.Cookie{
		Name:     "session",
		Value:    jwtString,
		Expires:  time.Now().Add(time.Minute * 15),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	}, nil
}

func createSessionCookie(user models.User, currentCredential *webauthn.Credential, sk ed25519.PrivateKey) (*http.Cookie, error) {
	var currentCredentialName string
	for name, credential := range user.WebauthnData.Credentials {
		if bytes.Equal(credential.ID, currentCredential.ID) {
			currentCredentialName = name
			break
		}
	}

	jwtString, err := common.CreateEnclaveJWT(user, currentCredentialName, sk)
	if err != nil {
		return nil, fmt.Errorf("failed to create jwt: %w", err)
	}

	return &http.Cookie{
		Name:     "session",
		Value:    jwtString,
		Expires:  time.Now().Add(time.Hour * 24),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	}, nil
}
