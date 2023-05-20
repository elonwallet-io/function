package models

import (
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type User struct {
	WebauthnData            WebauthnData                   `json:"webauthn_data"`
	Wallets                 Wallets                        `json:"wallets"`
	OTP                     OTP                            `json:"otp,omitempty"`
	Email                   string                         `json:"email"`
	Networks                []Network                      `json:"networks"`
	EmergencyAccessContacts map[string]EmergencyAccessData `json:"emergency_access_contacts"`
	EmergencyAccessGrantors map[string]EmergencyAccessData `json:"emergency_access_grantors"`
}

func NewUser(email string, displayName string) User {
	id, err := uuid.NewRandom()
	if err != nil {
		log.Fatal().Caller().Err(err).Msg("failed to generate unique user id")
	}

	return User{
		Email: email,
		WebauthnData: WebauthnData{
			ID:          id.String(),
			Name:        email,
			DisplayName: displayName,
			Credentials: make(map[string]webauthn.Credential),
			Sessions:    make(map[string]webauthn.SessionData),
		},
		Wallets:                 make(Wallets, 0),
		EmergencyAccessContacts: make(map[string]EmergencyAccessData),
		EmergencyAccessGrantors: make(map[string]EmergencyAccessData),
	}
}
