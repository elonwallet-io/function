package handlers

import (
	"fmt"
	"github.com/Leantar/elonwallet-function/config"
	"github.com/Leantar/elonwallet-function/models"
	"github.com/Leantar/elonwallet-function/server/common"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

const (
	RegistrationKey  = "registration"
	LoginKey         = "login"
	TransactionKey   = "transaction"
	AddCredentialKey = "add_credential"
)

type Api struct {
	w          *webauthn.WebAuthn
	repo       common.Repository
	signingKey models.SigningKey
}

func NewApi(cfg config.Config, repo common.Repository, signingKey models.SigningKey) (*Api, error) {
	w, err := webauthn.New(&webauthn.Config{
		RPID:          cfg.FrontendHost,
		RPDisplayName: "ElonWallet",
		RPOrigins:     []string{cfg.FrontendURL},
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			UserVerification: protocol.VerificationRequired,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create webauthn: %w", err)
	}

	return &Api{
		w:          w,
		repo:       repo,
		signingKey: signingKey,
	}, nil
}
