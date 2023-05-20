package handlers

import (
	"fmt"
	"github.com/Leantar/elonwallet-function/models"
	"github.com/Leantar/elonwallet-function/server/common"
	"github.com/labstack/echo/v4"
	"net/http"
	"time"
)

func (a *Api) HandleCreateEmergencyContact() echo.HandlerFunc {
	type input struct {
		Email               string `json:"contact_email" validate:"required,email"`
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
		enclaveURL, err := backendApiClient.GetEnclaveURL(in.Email)
		if err != nil {
			return err
		}

		enclaveApiClient := common.NewEnclaveApiClient(enclaveURL)
		err = enclaveApiClient.SendEmergencyAccessInvitation(in.WaitingPeriodInDays)
		if err != nil {
			return err
		}

		user.EmergencyAccessContacts[in.Email] = &models.EmergencyAccessData{
			Email:                in.Email,
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

		return c.NoContent(http.StatusCreated)
	}
}

func (a *Api) HandleGetEmergencyContacts() echo.HandlerFunc {
	type output struct {
		EmergencyContacts []models.EmergencyAccessData `json:"emergency_contacts"`
	}
	return func(c echo.Context) error {
		user := c.Get("user").(models.User)

		contacts := make([]models.EmergencyAccessData, len(user.EmergencyAccessContacts))
		i := 0
		for _, contact := range user.EmergencyAccessContacts {
			contacts[i] = *contact
			i++
		}

		return c.JSON(http.StatusOK, output{
			EmergencyContacts: contacts,
		})
	}
}

func (a *Api) HandleEmergencyAccessGrantResponse() echo.HandlerFunc {
	type input struct {
		Accept bool `json:"accept"`
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
		claims := c.Get("claims").(common.EnclaveClaims)

		data, ok := user.EmergencyAccessContacts[claims.Subject]
		if !ok {
			return echo.NewHTTPError(http.StatusNotFound)
		}

		if data.HasAccepted {
			return echo.NewHTTPError(http.StatusBadRequest, "Invitation has already been accepted")
		}

		backendApiClient, err := common.NewBackendApiClient(a.cfg.BackendURL, user, a.signingKey.PrivateKey)
		if err != nil {
			return fmt.Errorf("failed to create backend api client: %w", err)
		}

		var title string
		var body string
		if in.Accept {
			data.HasAccepted = true
			user.EmergencyAccessContacts[claims.Subject] = data
			title = "Invitation has been accepted"
			body = fmt.Sprintf("%s has accepted your request to be your emergency contact", claims.Subject)
		} else {
			delete(user.EmergencyAccessContacts, claims.Subject)
			title = "Invitation has been rejected"
			body = fmt.Sprintf("%s has rejected your request to be your emergency contact", claims.Subject)
		}
		err = backendApiClient.SendEmail(user.Email, title, body)
		if err != nil {
			return err
		}
		err = a.repo.UpsertUser(user)
		if err != nil {
			return err
		}

		return c.NoContent(http.StatusOK)
	}
}

func (a *Api) HandleEmergencyAccessRequest() echo.HandlerFunc {
	return func(c echo.Context) error {
		user := c.Get("user").(models.User)
		claims := c.Get("claims").(common.EnclaveClaims)

		data, ok := user.EmergencyAccessContacts[claims.Subject]
		if !ok {
			return echo.NewHTTPError(http.StatusNotFound)
		}

		if !data.HasAccepted {
			return echo.NewHTTPError(http.StatusBadRequest, "You must accept the invitation first")
		} else if data.HasRequestedTakeover {
			return echo.NewHTTPError(http.StatusBadRequest, "Takeover has already been requested")
		}

		data.HasRequestedTakeover = true
		data.TakeoverAllowedAfter = time.Now().Add(time.Duration(data.WaitingPeriodInDays) * 24 * time.Hour).Unix()
		err := a.repo.UpsertUser(user)
		if err != nil {
			return err
		}

		backendApiClient, err := common.NewBackendApiClient(a.cfg.BackendURL, user, a.signingKey.PrivateKey)
		if err != nil {
			return fmt.Errorf("failed to create backend api client: %w", err)
		}
		title := "Emergency Access to requested"
		body := fmt.Sprintf("%s has requested emergency access to your account. If you don't deny this request before %v, your account may be taken over.", claims.Subject, time.Unix(data.TakeoverAllowedAfter, 0))
		err = backendApiClient.SendEmail(user.Email, title, body)
		if err != nil {
			return err
		}

		return c.NoContent(http.StatusOK)
	}
}

func (a *Api) HandleEmergencyAccessTakeoverRequest() echo.HandlerFunc {
	type input struct {
		GrantorEmail string `json:"grantor_email" validate:"required,email"`
	}
	type output struct {
		Wallets []models.Wallet
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
		data, ok := user.EmergencyAccessContacts[in.GrantorEmail]
		if !ok {
			return echo.NewHTTPError(http.StatusNotFound)
		}

		if !data.HasRequestedTakeover {
			return echo.NewHTTPError(http.StatusBadRequest, "Emergency access must be requested first")
		} else if time.Now().Before(time.Unix(data.TakeoverAllowedAfter, 0)) {
			return echo.NewHTTPError(http.StatusBadRequest, "Waiting period is not yet over. Try again at a later time")
		}

		return c.JSON(http.StatusOK, output{
			Wallets: user.Wallets,
		})
	}
}
