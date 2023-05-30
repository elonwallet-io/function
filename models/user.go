package models

import (
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type User struct {
	WebauthnData            WebauthnData                       `json:"webauthn_data"`
	Wallets                 Wallets                            `json:"wallets"`
	OTP                     OTP                                `json:"otp,omitempty"`
	Email                   string                             `json:"email"`
	Networks                []Network                          `json:"networks"`
	EmergencyAccessContacts map[string]*EmergencyAccessContact `json:"emergency_access_contacts"`
	EmergencyAccessGrants   map[string]*EmergencyAccessGrant   `json:"emergency_access_grants"`
}

func NewUser(email string, displayName string) User {
	id, err := uuid.NewRandom()
	if err != nil {
		log.Fatal().Caller().Err(err).Msg("failed to generate unique user id")
	}

	return User{
		Email: email,
		WebauthnData: WebauthnData{
			ID:                  id.String(),
			Name:                email,
			DisplayName:         displayName,
			Credentials:         make(map[string]webauthn.Credential),
			Sessions:            make(map[string]webauthn.SessionData),
			PendingTransactions: make(map[string]TransactionParams),
		},
		Wallets:                 make(Wallets, 0),
		EmergencyAccessContacts: make(map[string]*EmergencyAccessContact),
		EmergencyAccessGrants:   make(map[string]*EmergencyAccessGrant),
	}
}
