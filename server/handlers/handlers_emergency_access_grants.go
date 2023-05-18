package handlers

import (
	"fmt"
	"github.com/Leantar/elonwallet-function/models"
	"github.com/Leantar/elonwallet-function/server/common"
	"github.com/labstack/echo/v4"
	"net/http"
	"time"
)

func (a *Api) HandleCreateEmergencyAccessGrant() echo.HandlerFunc {
	type input struct {
		GrantorEmail        string `json:"grantor_email" validate:"required,email"`
		WaitingPeriodInDays uint64 `json:"waiting_period_in_days" validate:"required"`
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

		backendApiClient, err := common.NewBackendApiClient(a.cfg.BackendURL, user, a.signingKey.PrivateKey)
		if err != nil {
			return fmt.Errorf("failed to create backend api client: %w", err)
		}
		enclaveURL, err := backendApiClient.GetEnclaveURL(in.GrantorEmail)
		if err != nil {
			return err
		}

		user.EmergencyAccessGrantors[in.GrantorEmail] = models.EmergencyAccessContact{
			Email:                in.GrantorEmail,
			EnclaveURL:           enclaveURL,
			HasAccepted:          false,
			HasRequestedTakeover: false,
			WaitingPeriodInDays:  in.WaitingPeriodInDays,
			TakeoverAllowedAfter: 0,
		}
		err = a.repo.UpsertUser(user)
		if err != nil {
			return err
		}

		err = backendApiClient.SendEmail(user.Email, "Emergency Access Invitation received", "You have received a pending invitation to be someones emergency contacts. Visit https://elonwallet.io/emergency-access to review it.")
		if err != nil {
			return err
		}

		return c.NoContent(http.StatusOK)
	}
}

func (a *Api) HandleGetEmergencyAccessGrantors() echo.HandlerFunc {
	type output struct {
		EmergencyAccessGrantors []models.EmergencyAccessContact `json:"emergency_access_grantors"`
	}
	return func(c echo.Context) error {
		user := c.Get("user").(models.User)

		grantors := make([]models.EmergencyAccessContact, len(user.EmergencyAccessGrantors))
		i := 0
		for _, data := range user.EmergencyAccessGrantors {
			grantors[i] = data
			i++
		}

		return c.JSON(http.StatusOK, output{
			EmergencyAccessGrantors: grantors,
		})
	}
}

func (a *Api) HandleRespondEmergencyAccessGrant() echo.HandlerFunc {
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
		data, ok := user.EmergencyAccessGrantors[in.GrantorEmail]
		if !ok {
			return echo.NewHTTPError(http.StatusNotFound, "Invitation not found")
		}

		if data.HasAccepted {
			return echo.NewHTTPError(http.StatusBadRequest, "Invitation has already been accepted")
		}

		enclaveApiClient := common.NewEnclaveApiClient(data.EnclaveURL, a.cfg.DevelopmentMode)
		err := enclaveApiClient.SendEmergencyAccessInvitationResponse(in.Accept)
		if err != nil {
			return err
		}

		if in.Accept {
			data.HasAccepted = true
			user.EmergencyAccessGrantors[in.GrantorEmail] = data
		} else {
			delete(user.EmergencyAccessGrantors, in.GrantorEmail)
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
		data, ok := user.EmergencyAccessGrantors[in.GrantorEmail]
		if !ok {
			return echo.NewHTTPError(http.StatusNotFound)
		}

		if !data.HasAccepted {
			return echo.NewHTTPError(http.StatusBadRequest, "You must accept the invitation first")
		} else if data.HasRequestedTakeover {
			return echo.NewHTTPError(http.StatusBadRequest, "Takeover has already been requested")
		}

		enclaveApiClient := common.NewEnclaveApiClient(data.EnclaveURL, a.cfg.DevelopmentMode)
		err := enclaveApiClient.SendEmergencyAccessRequest()
		if err != nil {
			return err
		}

		data.HasRequestedTakeover = true
		data.TakeoverAllowedAfter = time.Now().Add(time.Duration(data.WaitingPeriodInDays) * 24 * time.Hour).Unix()
		user.EmergencyAccessGrantors[in.GrantorEmail] = data
		err = a.repo.UpsertUser(user)
		if err != nil {
			return err
		}

		return c.NoContent(http.StatusOK)
	}
}

func (a *Api) HandleEmergencyAccessGrantRevocation() echo.HandlerFunc {
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
		_, ok := user.EmergencyAccessGrantors[in.GrantorEmail]
		if !ok {
			return echo.NewHTTPError(http.StatusNotFound)
		}

		delete(user.EmergencyAccessGrantors, in.GrantorEmail)
		err := a.repo.UpsertUser(user)
		if err != nil {
			return err
		}

		backendApiClient := common.NewBackendApiClient(a.cfg.BackendURL, a.cfg.DevelopmentMode)
		err = backendApiClient.SendEmail(user.Email, "Emergency Access Grant was revoked", fmt.Sprintf("You are no longer registered as an emergency contact for the user %s", in.GrantorEmail))
		if err != nil {
			return err
		}

		return c.NoContent(http.StatusOK)
	}
}
func (a *Api) HandleEmergencyAccessRequestDenial() echo.HandlerFunc {
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
		data, ok := user.EmergencyAccessGrantors[in.GrantorEmail]
		if !ok {
			return echo.NewHTTPError(http.StatusNotFound)
		}

		data.HasRequestedTakeover = false
		data.TakeoverAllowedAfter = 0
		err := a.repo.UpsertUser(user)
		if err != nil {
			return err
		}

		backendApiClient := common.NewBackendApiClient(a.cfg.BackendURL, a.cfg.DevelopmentMode)
		err = backendApiClient.SendEmail(user.Email, "Emergency Access Request was denied", fmt.Sprintf("Your pending request to takeover the wallets of %s has been denied by the owner", in.GrantorEmail))
		if err != nil {
			return err
		}

		return c.NoContent(http.StatusOK)
	}
}
