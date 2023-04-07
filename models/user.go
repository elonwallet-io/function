package models

import (
	"crypto/ed25519"
	"crypto/rand"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type User struct {
	WebauthnData  WebauthnData       `json:"webauthn_data"`
	Wallets       Wallets            `json:"wallets"`
	JWTSigningKey ed25519.PrivateKey `json:"jwt_signing_key"`
	Email         string             `json:"email"`
}

func NewUser(name string, displayName string) User {
	id, err := uuid.NewRandom()
	if err != nil {
		log.Fatal().Caller().Err(err).Msg("failed to generate unique user id")
	}

	_, sk, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatal().Caller().Err(err).Msg("failed to generate jwt signing key")
	}

	return User{
		WebauthnData: WebauthnData{
			ID:          id.String(),
			Name:        name,
			DisplayName: displayName,
			Credentials: make(map[string]webauthn.Credential),
			Sessions:    make(map[string]webauthn.SessionData),
		},
		Wallets:       make(Wallets, 0),
		JWTSigningKey: sk,
	}
}
