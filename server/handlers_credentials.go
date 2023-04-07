package server

import (
	"github.com/Leantar/elonwallet-function/models"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/protocol/webauthncose"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"net/http"
)

func (a *Api) CreateCredentialInitialize() echo.HandlerFunc {
	return func(c echo.Context) error {
		user := c.Get("user").(*models.User)

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
			credCreationOpts.CredentialExcludeList = user.WebauthnData.CredentialExcludeList()
		}

		options, session, err := a.w.BeginRegistration(user.WebauthnData, registerOptions)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		user.WebauthnData.Sessions[AddCredentialKey] = *session

		return c.JSON(http.StatusOK, options)
	}
}

func (a *Api) CreateCredentialFinalize() echo.HandlerFunc {
	type input struct {
		CredentialName   string                              `json:"name" validate:"required,alphanum"`
		CreationResponse protocol.CredentialCreationResponse `json:"creation_response"`
	}
	return func(c echo.Context) error {
		user := c.Get("user").(*models.User)

		var in input
		if err := c.Bind(&in); err != nil {
			return err
		}
		if err := c.Validate(&in); err != nil {
			return err
		}

		_, ok := user.WebauthnData.Credentials[in.CredentialName]
		if ok {
			return echo.NewHTTPError(http.StatusBadRequest, "a credential with this name already exists")
		}

		session, ok := user.WebauthnData.Sessions[AddCredentialKey]
		if !ok {
			return echo.NewHTTPError(http.StatusBadRequest, "CreateCredential must be initialized beforehand")
		}
		delete(user.WebauthnData.Sessions, AddCredentialKey)

		ccr, err := in.CreationResponse.Parse()
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		cred, err := a.w.CreateCredential(user.WebauthnData, session, ccr)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		user.WebauthnData.Credentials[in.CredentialName] = *cred

		return c.NoContent(http.StatusOK)
	}
}

func (a *Api) RemoveCredential() echo.HandlerFunc {
	return func(c echo.Context) error {
		user := c.Get("user").(*models.User)
		claims := c.Get("claims").(jwt.MapClaims)
		credName := c.Param("name")

		_, ok := user.WebauthnData.Credentials[credName]
		if !ok {
			return echo.NewHTTPError(http.StatusNotFound, "Credential does not exist")
		}

		loginCred := claims["credential"].(string)
		if loginCred == credName {
			return echo.NewHTTPError(http.StatusBadRequest, "You cannot delete the credential you are currently logged in with")
		}

		delete(user.WebauthnData.Credentials, credName)

		return c.NoContent(http.StatusOK)
	}
}

func (a *Api) GetCredentials() echo.HandlerFunc {
	type credential struct {
		Name          string `json:"name"`
		CurrentlyUsed bool   `json:"currently_used"`
	}
	type output struct {
		Credentials []credential `json:"credentials"`
	}
	return func(c echo.Context) error {
		user := c.Get("user").(*models.User)
		claims := c.Get("claims").(jwt.MapClaims)
		currentCred := claims["credential"].(string)

		credentials := make([]credential, len(user.WebauthnData.Credentials))
		i := 0
		for name := range user.WebauthnData.Credentials {
			credentials[i] = credential{
				Name:          name,
				CurrentlyUsed: name == currentCred,
			}
			i++
		}

		return c.JSON(http.StatusOK, output{
			Credentials: credentials,
		})
	}
}
