package handlers

import (
	"fmt"
	"github.com/Leantar/elonwallet-function/models"
	"github.com/Leantar/elonwallet-function/server/common"
	"github.com/labstack/echo/v4"
	"net/http"
	"time"
)

func (a *Api) HandleEmergencyAccessGrantInvitation() echo.HandlerFunc {
	return func(c echo.Context) error {
		user := c.Get("user").(models.User)
		claims := c.Get("claims").(common.EnclaveClaims)

		backendApiClient, err := common.NewBackendApiClient(a.cfg.BackendURL, user, a.signingKey.PrivateKey)
		if err != nil {
			return fmt.Errorf("failed to create backend api client: %w", err)
		}
		enclaveURL, err := backendApiClient.GetEnclaveURL(claims.Subject)
		if err != nil {
			return err
		}

		user.EmergencyAccessGrants[claims.Subject] = &models.EmergencyAccessGrant{
			Email:      claims.Subject,
			EnclaveURL: enclaveURL,
		}
		err = a.repo.UpsertUser(user)
		if err != nil {
			return err
		}

		return c.NoContent(http.StatusOK)
	}
}

func (a *Api) HandleGetEmergencyAccessGrants() echo.HandlerFunc {
	type output struct {
		EmergencyAccessGrants []models.EmergencyAccessGrant `json:"emergency_access_grants"`
	}
	return func(c echo.Context) error {
		user := c.Get("user").(models.User)

		grants := make([]models.EmergencyAccessGrant, len(user.EmergencyAccessGrants))
		i := 0
		for _, data := range user.EmergencyAccessGrants {
			grants[i] = *data
			i++
		}

		return c.JSON(http.StatusOK, output{
			EmergencyAccessGrants: grants,
		})
	}
}

func (a *Api) HandleRespondEmergencyAccessGrantInvitation() echo.HandlerFunc {
	type input struct {
		GrantorEmail string `json:"grantor_email" validate:"required,email"`
		Accept       bool   `json:"accept"`
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

		data, ok := user.EmergencyAccessGrants[in.GrantorEmail]
		if !ok {
			return echo.NewHTTPError(http.StatusNotFound)
		}

		if data.HasAccepted {
			return echo.NewHTTPError(http.StatusBadRequest, "Invitation has already been accepted")
		}

		enclaveApiClient, err := common.NewEnclaveApiClient(data.EnclaveURL, user, a.signingKey.PrivateKey)
		if err != nil {
			return fmt.Errorf("failed to create enclave api client: %w", err)
		}

		err = enclaveApiClient.RespondEmergencyAccessInvitation(in.Accept)
		if err != nil {
			return err
		}

		if in.Accept {
			data.HasAccepted = true
		} else {
			delete(user.EmergencyAccessGrants, in.GrantorEmail)
		}
		err = a.repo.UpsertUser(user)
		if err != nil {
			return err
		}

		return c.NoContent(http.StatusOK)
	}
}

func (a *Api) HandleRequestEmergencyAccess() echo.HandlerFunc {
	type input struct {
		GrantorEmail string `json:"grantor_email" validate:"required,email"`
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

		data, ok := user.EmergencyAccessGrants[in.GrantorEmail]
		if !ok {
			return echo.NewHTTPError(http.StatusNotFound)
		}

		if !data.HasAccepted {
			return echo.NewHTTPError(http.StatusBadRequest, "You must accept the invitation first")
		} else if data.HasRequestedTakeover {
			return echo.NewHTTPError(http.StatusBadRequest, "Takeover has already been requested")
		}

		enclaveApiClient, err := common.NewEnclaveApiClient(data.EnclaveURL, user, a.signingKey.PrivateKey)
		if err != nil {
			return fmt.Errorf("failed to create enclave api client: %w", err)
		}

		takeoverAllowedAfter, err := enclaveApiClient.RequestEmergencyAccess()
		if err != nil {
			return err
		}

		data.HasRequestedTakeover = true
		data.TakeoverAllowedAfter = takeoverAllowedAfter
		err = a.repo.UpsertUser(user)
		if err != nil {
			return err
		}

		return c.NoContent(http.StatusOK)
	}
}

func (a *Api) HandleEmergencyAccessGrantRemoval() echo.HandlerFunc {
	return func(c echo.Context) error {
		user := c.Get("user").(models.User)
		claims := c.Get("claims").(common.EnclaveClaims)

		_, ok := user.EmergencyAccessGrants[claims.Subject]
		if !ok {
			return echo.NewHTTPError(http.StatusNotFound)
		}

		delete(user.EmergencyAccessGrants, claims.Subject)
		err := a.repo.UpsertUser(user)
		if err != nil {
			return err
		}

		return c.NoContent(http.StatusOK)
	}
}

func (a *Api) HandleEmergencyAccessRequestDenial() echo.HandlerFunc {
	return func(c echo.Context) error {
		user := c.Get("user").(models.User)
		claims := c.Get("claims").(common.EnclaveClaims)

		data, ok := user.EmergencyAccessGrants[claims.Subject]
		if !ok {
			return echo.NewHTTPError(http.StatusNotFound)
		}

		backendApiClient, err := common.NewBackendApiClient(a.cfg.BackendURL, user, a.signingKey.PrivateKey)
		if err != nil {
			return fmt.Errorf("failed to create backend api client: %w", err)
		}

		title := "Emergency Access Request was denied"
		body := fmt.Sprintf("Your pending request to takeover the account of %s has been denied", claims.Subject)
		err = backendApiClient.SendNotification(title, body)
		if err != nil {
			return err
		}

		data.HasRequestedTakeover = false
		data.TakeoverAllowedAfter = 0
		err = a.repo.UpsertUser(user)
		if err != nil {
			return err
		}

		return c.NoContent(http.StatusOK)
	}
}

func (a *Api) HandleRequestEmergencyAccessTakeover() echo.HandlerFunc {
	type input struct {
		GrantorEmail string `json:"grantor_email" validate:"required,email"`
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
		data, ok := user.EmergencyAccessGrants[in.GrantorEmail]
		if !ok {
			return echo.NewHTTPError(http.StatusNotFound)
		}

		if !data.HasRequestedTakeover {
			return echo.NewHTTPError(http.StatusBadRequest, "Emergency access must be requested first")
		} else if time.Now().Before(time.Unix(data.TakeoverAllowedAfter, 0)) {
			return echo.NewHTTPError(http.StatusBadRequest, "Waiting period is not yet over. Try again at a later time")
		}

		enclaveApiClient, err := common.NewEnclaveApiClient(data.EnclaveURL, user, a.signingKey.PrivateKey)
		if err != nil {
			return fmt.Errorf("failed to create enclave api client: %w", err)
		}

		wallets, err := enclaveApiClient.RequestEmergencyAccessTakeover()
		if err != nil {
			return err
		}

		for _, wallet := range wallets {
			wallet.Public = false
			wallet.Name = fmt.Sprintf("%s (%s)", wallet.Name, in.GrantorEmail)
			user.Wallets = append(user.Wallets, wallet)
		}

		err = a.repo.UpsertUser(user)
		if err != nil {
			return err
		}

		return c.NoContent(http.StatusOK)
	}
}
