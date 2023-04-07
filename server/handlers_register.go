package server

import (
	"fmt"
	"github.com/Leantar/elonwallet-function/models"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/protocol/webauthncose"
	"github.com/labstack/echo/v4"
	"net/http"
	"net/url"
)

func (a *Api) RegisterInitialize() echo.HandlerFunc {
	type input struct {
		Email string `validate:"required,email"`
	}
	return func(c echo.Context) error {
		email, err := url.QueryUnescape(c.QueryParam("email"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid escape sequence").SetInternal(err)
		}
		in := input{
			Email: email,
		}
		if err := c.Validate(&in); err != nil {
			return err
		}

		user := models.NewUser(email, email)
		registerOptions := func(credCreationOpts *protocol.PublicKeyCredentialCreationOptions) {
			credCreationOpts.Parameters = []protocol.CredentialParameter{
				{
					Type:      protocol.PublicKeyCredentialType,
					Algorithm: webauthncose.AlgEdDSA,
				},
				{
					Type:      protocol.PublicKeyCredentialType,
					Algorithm: webauthncose.AlgES256,
				},
				{
					Type:      protocol.PublicKeyCredentialType,
					Algorithm: webauthncose.AlgRS256,
				},
			}
			credCreationOpts.Attestation = protocol.PreferNoAttestation
		}

		options, session, err := a.w.BeginRegistration(user.WebauthnData, registerOptions)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		user.WebauthnData.Sessions[RegistrationKey] = *session
		if err := a.d.SaveUser(user); err != nil {
			return fmt.Errorf("failed to save user: %w", err)
		}

		return c.JSON(http.StatusOK, options)
	}
}

func (a *Api) RegisterFinalize() echo.HandlerFunc {
	return func(c echo.Context) error {
		user := c.Get("user").(*models.User)
		session, ok := user.WebauthnData.Sessions[RegistrationKey]
		if !ok {
			return echo.NewHTTPError(http.StatusBadRequest, "registration must be initialized beforehand")
		}
		delete(user.WebauthnData.Sessions, RegistrationKey)

		cred, err := a.w.FinishRegistration(user.WebauthnData, session, c.Request())
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		user.WebauthnData.Credentials["Default"] = *cred

		//Create a default wallet for the user
		wallet, err := models.NewWallet("Default", false)
		if err != nil {
			return fmt.Errorf("failed to create new wallet: %w", err)
		}
		user.Wallets = append(user.Wallets, wallet)

		return c.NoContent(http.StatusOK)
	}
}
