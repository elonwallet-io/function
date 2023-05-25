package handlers

import (
	"errors"
	"fmt"
	"github.com/Leantar/elonwallet-function/models"
	"github.com/Leantar/elonwallet-function/server/common"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/labstack/echo/v4"
	"net/http"
)

func (a *Api) HandleRegisterInitialize() echo.HandlerFunc {
	type input struct {
		Email string `query:"email" validate:"required,email"`
	}
	return func(c echo.Context) error {
		var in input
		if err := c.Bind(&in); err != nil {
			return err
		}
		if err := c.Validate(&in); err != nil {
			return err
		}

		user, err := a.repo.GetUser()
		if len(user.WebauthnData.Credentials) > 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "User is already registered")
		} else if err != nil && !errors.Is(err, common.ErrNotFound) {
			return fmt.Errorf("failed to check if user exists: %w", err)
		}

		user = models.NewUser(in.Email, in.Email)
		registrationOptions := getCreationOptions(nil)

		options, session, err := a.w.BeginRegistration(user.WebauthnData, registrationOptions)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		user.WebauthnData.Sessions[RegistrationKey] = *session
		if err := a.repo.UpsertUser(user); err != nil {
			return err
		}

		return c.JSON(http.StatusOK, options)
	}
}

func (a *Api) HandleRegisterFinalize() echo.HandlerFunc {
	type input struct {
		CredentialName   string                              `json:"name" validate:"required,alphanum"`
		CreationResponse protocol.CredentialCreationResponse `json:"creation_response"`
	}
	return func(c echo.Context) error {
		var in input
		if err := c.Bind(&in); err != nil {
			return err
		}
		if err := c.Validate(&in); err != nil {
			return err
		}

		user, err := a.repo.GetUser()
		if err != nil {
			return err
		}

		session, ok := user.WebauthnData.Sessions[RegistrationKey]
		if !ok {
			return echo.NewHTTPError(http.StatusBadRequest, "Registration must be initialized beforehand")
		}
		delete(user.WebauthnData.Sessions, RegistrationKey)

		ccr, err := in.CreationResponse.Parse()
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		cred, err := a.w.CreateCredential(user.WebauthnData, session, ccr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		user.WebauthnData.Credentials[in.CredentialName] = *cred

		wallet, err := a.createWallet("Default", true, user)
		if err != nil {
			return err
		}
		user.Wallets = append(user.Wallets, wallet)

		err = a.repo.UpsertUser(user)
		if err != nil {
			return err
		}

		return c.NoContent(http.StatusOK)
	}
}
