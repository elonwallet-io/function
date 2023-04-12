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

func (a *Api) RegisterInitialize() echo.HandlerFunc {
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
			return echo.NewHTTPError(http.StatusBadRequest, "user is already registered")
		} else if err != nil && !errors.Is(err, common.ErrNotFound) {
			return fmt.Errorf("failed to check if user exists: %w", err)
		}

		user = models.NewUser(in.Email, in.Email)
		registrationOptions := common.GetCreationOptions(nil)

		options, session, err := a.w.BeginRegistration(user.WebauthnData, registrationOptions)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		user.WebauthnData.Sessions[RegistrationKey] = *session
		if err := a.repo.UpsertUser(user); err != nil {
			return fmt.Errorf("failed to save user: %w", err)
		}

		return c.JSON(http.StatusOK, options)
	}
}

func (a *Api) RegisterFinalize() echo.HandlerFunc {
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
			return fmt.Errorf("failed to get user: %w", err)
		}

		session, ok := user.WebauthnData.Sessions[RegistrationKey]
		if !ok {
			return echo.NewHTTPError(http.StatusBadRequest, "registration must be initialized beforehand")
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

		//Create a default wallet for the user
		wallet, err := models.NewWallet("Default", false)
		if err != nil {
			return fmt.Errorf("failed to create new wallet: %w", err)
		}
		user.Wallets = append(user.Wallets, wallet)

		err = a.repo.UpsertUser(user)
		if err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}

		return c.NoContent(http.StatusOK)
	}
}
