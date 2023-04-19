package handlers

import (
	"crypto/rand"
	"fmt"
	"github.com/Leantar/elonwallet-function/models"
	"github.com/Leantar/elonwallet-function/server/common"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/labstack/echo/v4"
	"math/big"
	"net/http"
	"strings"
	"time"
)

func (a *Api) CreateCredentialInitialize() echo.HandlerFunc {
	return func(c echo.Context) error {
		user, err := a.repo.GetUser()
		if err != nil {
			return fmt.Errorf("failed to get user: %w", err)
		}

		registrationOptions := common.GetCreationOptions(user.WebauthnData.CredentialExcludeList())

		options, session, err := a.w.BeginRegistration(user.WebauthnData, registrationOptions)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		user.WebauthnData.Sessions[AddCredentialKey] = *session
		err = a.repo.UpsertUser(user)
		if err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}

		return c.JSON(http.StatusOK, options)
	}
}

func (a *Api) CreateCredentialFinalize() echo.HandlerFunc {
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
			return fmt.Errorf("failed to update user: %w", err)
		}

		return c.NoContent(http.StatusOK)
	}
}

func (a *Api) RemoveCredential() echo.HandlerFunc {
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

		user, err := a.repo.GetUser()
		if err != nil {
			return fmt.Errorf("failed to get user: %w", err)
		}

		_, ok := user.WebauthnData.Credentials[in.CredentialName]
		if !ok {
			return echo.NewHTTPError(http.StatusNotFound, "Credential does not exist")
		}

		if claims.Credential == in.CredentialName {
			return echo.NewHTTPError(http.StatusBadRequest, "You cannot delete the credential you are currently logged in with")
		}
		delete(user.WebauthnData.Credentials, in.CredentialName)

		err = a.repo.UpsertUser(user)
		if err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}

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
		claims := c.Get("claims").(common.EnclaveClaims)

		user, err := a.repo.GetUser()
		if err != nil {
			return fmt.Errorf("failed to get user: %w", err)
		}

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

func (a *Api) CreateOTP() echo.HandlerFunc {
	type output struct {
		OTP string `json:"otp"`
	}
	return func(c echo.Context) error {
		user := c.Get("user").(models.User)

		otp, err := generateOTP()
		if err != nil {
			return fmt.Errorf("failed to generate otp: %w", err)
		}

		user.OTP = models.OTP{
			Secret:     otp,
			ValidUntil: time.Now().Add(time.Minute * 30).Unix(),
			TimesTried: 0,
		}

		return c.JSON(http.StatusCreated, output{OTP: ""})
	}
}

func generateOTP() (string, error) {
	var charset = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	var charsetLength = new(big.Int).SetInt64(int64(len(charset)))

	var sb strings.Builder
	for i := 0; i < 17; i++ {
		if i == 5 || i == 11 {
			sb.WriteString("-")
		} else {
			index, err := rand.Int(rand.Reader, charsetLength)
			if err != nil {
				return "", fmt.Errorf("failed to generate random char: %w", err)
			}
			sb.WriteRune(charset[index.Int64()])
		}
	}

	return sb.String(), nil
}
