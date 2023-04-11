package models

import (
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type User struct {
	WebauthnData WebauthnData `json:"webauthn_data"`
	Wallets      Wallets      `json:"wallets"`
	Email        string       `json:"email"`
}

func NewUser(name string, displayName string) User {
	id, err := uuid.NewRandom()
	if err != nil {
		log.Fatal().Caller().Err(err).Msg("failed to generate unique user id")
	}

	return User{
		Email: name,
		WebauthnData: WebauthnData{
			ID:          id.String(),
			Name:        name,
			DisplayName: displayName,
			Credentials: make(map[string]webauthn.Credential),
			Sessions:    make(map[string]webauthn.SessionData),
		},
		Wallets: make(Wallets, 0),
	}
}
