package handlers

import (
	"github.com/Leantar/elonwallet-function/models"
	"github.com/Leantar/elonwallet-function/server/common"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/labstack/echo/v4"
	"net/http"
)

func (a *Api) HandleCreateCredentialInitialize() echo.HandlerFunc {
	return func(c echo.Context) error {
		user := c.Get("user").(models.User)

		registrationOptions := getCreationOptions(user.WebauthnData.CredentialExcludeList())

		options, session, err := a.w.BeginRegistration(user.WebauthnData, registrationOptions)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		user.WebauthnData.Sessions[AddCredentialKey] = *session
		err = a.repo.UpsertUser(user)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, options)
	}
}

func (a *Api) HandleCreateCredentialFinalize() echo.HandlerFunc {
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

		user := c.Get("user").(models.User)

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
		err = a.repo.UpsertUser(user)
		if err != nil {
			return err
		}

		return c.NoContent(http.StatusOK)
	}
}

func (a *Api) HandleRemoveCredential() echo.HandlerFunc {
	type input struct {
		CredentialName string `param:"name" validate:"required,alphanum"`
	}
	return func(c echo.Context) error {
		var in input
		if err := c.Bind(&in); err != nil {
			return err
		}
		if err := c.Validate(&in); err != nil {
			return err
		}

		claims := c.Get("claims").(common.EnclaveClaims)
		user := c.Get("user").(models.User)

		_, ok := user.WebauthnData.Credentials[in.CredentialName]
		if !ok {
			return echo.NewHTTPError(http.StatusNotFound)
		}

		if claims.Credential == in.CredentialName {
			return echo.NewHTTPError(http.StatusBadRequest, "You cannot delete the credential you are currently logged in with")
		}
		delete(user.WebauthnData.Credentials, in.CredentialName)

		err := a.repo.UpsertUser(user)
		if err != nil {
			return err
		}

		return c.NoContent(http.StatusOK)
	}
}

func (a *Api) HandleGetCredentials() echo.HandlerFunc {
	type credential struct {
		Name          string `json:"name"`
		CurrentlyUsed bool   `json:"currently_used"`
	}
	type output struct {
		Credentials []credential `json:"credentials"`
	}
	return func(c echo.Context) error {
		claims := c.Get("claims").(common.EnclaveClaims)
		user := c.Get("user").(models.User)

		credentials := make([]credential, len(user.WebauthnData.Credentials))
		i := 0
		for name := range user.WebauthnData.Credentials {
			credentials[i] = credential{
				Name:          name,
				CurrentlyUsed: name == claims.Credential,
			}
			i++
		}

		return c.JSON(http.StatusOK, output{
			Credentials: credentials,
		})
	}
}
