package handlers

import (
	"crypto/ed25519"
	"fmt"
	"github.com/Leantar/elonwallet-function/config"
	"github.com/Leantar/elonwallet-function/models"
	"github.com/Leantar/elonwallet-function/server/common"
	"github.com/labstack/echo/v4"
	"net/http"
	"time"
)

func (a *Api) HandleCreateEmergencyContact() echo.HandlerFunc {
	type input struct {
		Email               string `json:"contact_email" validate:"required,email"`
		WaitingPeriodInDays uint64 `json:"waiting_period_in_days" validate:"required,gte=7,lt=100"`
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

		if in.Email == user.Email {
			return echo.NewHTTPError(http.StatusBadRequest, "You cannot add yourself as an emergency contact")
		}

		_, ok := user.EmergencyAccessContacts[in.Email]
		if ok {
			return echo.NewHTTPError(http.StatusConflict, "Contact already exists")
		}

		backendApiClient, err := common.NewBackendApiClient(a.cfg.BackendURL, user, a.signingKey.PrivateKey)
		if err != nil {
			return fmt.Errorf("failed to create backend api client: %w", err)
		}
		enclaveURL, err := backendApiClient.GetEnclaveURL(in.Email)
		if err != nil {
			return err
		}

		enclaveApiClient, err := common.NewEnclaveApiClient(enclaveURL, user, a.signingKey.PrivateKey)
		if err != nil {
			return fmt.Errorf("failed to create enclave api client: %w", err)
		}
		err = enclaveApiClient.InviteEmergencyAccessContact()
		if err != nil {
			return err
		}

		user.EmergencyAccessContacts[in.Email] = &models.EmergencyAccessContact{
			Email:               in.Email,
			EnclaveURL:          enclaveURL,
			WaitingPeriodInDays: in.WaitingPeriodInDays,
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
		EmergencyContacts []models.EmergencyAccessContact `json:"emergency_contacts"`
	}
	return func(c echo.Context) error {
		user := c.Get("user").(models.User)

		contacts := make([]models.EmergencyAccessContact, len(user.EmergencyAccessContacts))
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

func (a *Api) HandleRemoveEmergencyContact() echo.HandlerFunc {
	type input struct {
		Email string `param:"email" validate:"required,email"`
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

		data, ok := user.EmergencyAccessContacts[in.Email]
		if !ok {
			return echo.NewHTTPError(http.StatusNotFound)
		}

		enclaveApiClient, err := common.NewEnclaveApiClient(data.EnclaveURL, user, a.signingKey.PrivateKey)
		if err != nil {
			return fmt.Errorf("failed to create enclave api client: %w", err)
		}

		err = enclaveApiClient.RemoveEmergencyAccessGrant()
		if err != nil {
			return fmt.Errorf("failed to remove emergency access grant: %w", err)
		}

		if data.HasRequestedTakeover && data.NotificationSeriesID != "" {
			backendApiClient, err := common.NewBackendApiClient(a.cfg.BackendURL, user, a.signingKey.PrivateKey)
			if err != nil {
				return fmt.Errorf("failed to create backend api client: %w", err)
			}

			err = backendApiClient.DeleteNotificationSeries(data.NotificationSeriesID)
			if err != nil {
				return fmt.Errorf("failed to delete scheduled notifications: %w", err)
			}
		}

		delete(user.EmergencyAccessContacts, in.Email)
		err = a.repo.UpsertUser(user)
		if err != nil {
			return err
		}

		return c.NoContent(http.StatusOK)
	}
}

func (a *Api) HandleDenyEmergencyContactAccessRequest() echo.HandlerFunc {
	type input struct {
		Email string `param:"email" validate:"required,email"`
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

		data, ok := user.EmergencyAccessContacts[in.Email]
		if !ok {
			return echo.NewHTTPError(http.StatusNotFound)
		}

		enclaveApiClient, err := common.NewEnclaveApiClient(data.EnclaveURL, user, a.signingKey.PrivateKey)
		if err != nil {
			return fmt.Errorf("failed to create enclave api client: %w", err)
		}

		err = enclaveApiClient.DenyEmergencyAccessRequest()
		if err != nil {
			return fmt.Errorf("failed to deny emergency access request: %w", err)
		}

		backendApiClient, err := common.NewBackendApiClient(a.cfg.BackendURL, user, a.signingKey.PrivateKey)
		if err != nil {
			return fmt.Errorf("failed to create backend api client: %w", err)
		}

		err = backendApiClient.DeleteNotificationSeries(data.NotificationSeriesID)
		if err != nil {
			return fmt.Errorf("failed to delete scheduled notifications: %w", err)
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

		if in.Accept {
			data.HasAccepted = true
		} else {
			delete(user.EmergencyAccessContacts, claims.Subject)
		}

		err := a.repo.UpsertUser(user)
		if err != nil {
			return err
		}

		return c.NoContent(http.StatusOK)
	}
}

func (a *Api) HandleEmergencyContactAccessRequest() echo.HandlerFunc {
	type output struct {
		TakeoverAllowedAfter int64 `json:"takeover_allowed_after"`
	}
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

		backendApiClient, err := common.NewBackendApiClient(a.cfg.BackendURL, user, a.signingKey.PrivateKey)
		if err != nil {
			return fmt.Errorf("failed to create backend api client: %w", err)
		}

		notifications := createScheduledNotifications(data.WaitingPeriodInDays, claims.Subject, data.TakeoverAllowedAfter)
		seriesID, err := backendApiClient.ScheduleNotificationSeries(notifications)
		if err != nil {
			return err
		}

		data.NotificationSeriesID = seriesID
		err = a.repo.UpsertUser(user)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, output{data.TakeoverAllowedAfter})
	}
}

func (a *Api) HandleEmergencyContactTakeoverRequest() echo.HandlerFunc {
	type output struct {
		JWT     string          `json:"jwt"`
		Wallets []models.Wallet `json:"wallets"`
	}
	return func(c echo.Context) error {
		user := c.Get("user").(models.User)
		claims := c.Get("claims").(common.EnclaveClaims)

		data, ok := user.EmergencyAccessContacts[claims.Subject]
		if !ok {
			return echo.NewHTTPError(http.StatusNotFound)
		}

		if !data.HasRequestedTakeover {
			return echo.NewHTTPError(http.StatusBadRequest, "Emergency access must be requested first")
		} else if time.Now().Before(time.Unix(data.TakeoverAllowedAfter, 0)) {
			return echo.NewHTTPError(http.StatusBadRequest, "Waiting period is not yet over. Try again at a later time")
		}

		err := handleNotificationsOnTakeover(a.cfg, user, data, a.signingKey.PrivateKey)
		if err != nil {
			return err
		}

		err = removeEmergencyContacts(user, claims.Subject, a.repo, a.signingKey.PrivateKey)
		if err != nil {
			return fmt.Errorf("failed to remove all emergency contacts %w", err)
		}

		jwt, err := common.CreateBackendJWT(user, common.ScopeEnclave, a.signingKey.PrivateKey)
		if err != nil {
			return fmt.Errorf("failed to create jwt: %w", err)
		}

		return c.JSON(http.StatusOK, output{
			Wallets: user.Wallets,
			JWT:     jwt,
		})
	}
}

func handleNotificationsOnTakeover(cfg config.Config, user models.User, data *models.EmergencyAccessContact, sk ed25519.PrivateKey) error {
	backendApiClient, err := common.NewBackendApiClient(cfg.BackendURL, user, sk)
	if err != nil {
		return fmt.Errorf("failed to create backend api client: %w", err)
	}

	err = backendApiClient.DeleteNotificationSeries(data.NotificationSeriesID)
	if err != nil {
		return fmt.Errorf("failed to delete scheduled notifications: %w", err)
	}

	title := "Your Account was taken over"
	body := fmt.Sprintf("Your Account has been taken over by your emergency contact %s. Your remaining data will be deleted soon", data.Email)
	err = backendApiClient.SendNotification(title, body)
	if err != nil {
		return fmt.Errorf("failed to delete scheduled notifications: %w", err)
	}

	return nil
}

func createScheduledNotifications(waitingPeriodInDays uint64, contactEmail string, takeoverAllowedAfter int64) []models.ScheduledNotification {
	now := time.Now()
	takeoverTime := time.Unix(takeoverAllowedAfter, 0)

	notifications := make([]models.ScheduledNotification, waitingPeriodInDays)
	notifications[0] = models.ScheduledNotification{
		SendAfter: now.Unix(),
		Title:     "Emergency Access has been requested",
		Body: fmt.Sprintf(
			"%s has requested emergency access to your account. If you don't deny this request before %s, your account may be taken over.",
			contactEmail,
			takeoverTime.Format(time.RFC1123Z),
		),
	}

	for i := 1; i < int(waitingPeriodInDays); i++ {
		notifications[i] = models.ScheduledNotification{
			SendAfter: now.Add(time.Duration(i) * 24 * time.Hour).Unix(),
			Title:     "Emergency Access is pending",
			Body:      fmt.Sprintf("Your account may be taken over by %s on %s. Deny this request on https://elonwallet.io/emergency-access before it is too late.", contactEmail, takeoverTime.Format(time.RFC1123Z)),
		}
	}

	return notifications
}

func removeEmergencyContacts(user models.User, subject string, repo common.Repository, sk ed25519.PrivateKey) error {
	for _, contact := range user.EmergencyAccessContacts {
		if contact.Email == subject {
			continue
		}

		enclaveApiClient, err := common.NewEnclaveApiClient(contact.EnclaveURL, user, sk)
		if err != nil {
			return fmt.Errorf("failed to create enclave api client: %w", err)
		}

		err = enclaveApiClient.RemoveEmergencyAccessGrant()
		if err != nil {
			return fmt.Errorf("failed to remove emergency contact: %w", err)
		}
	}

	user.EmergencyAccessContacts = make(map[string]*models.EmergencyAccessContact, 0)
	return repo.UpsertUser(user)
}
