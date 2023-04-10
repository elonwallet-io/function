package server

import (
	"fmt"
	"github.com/Leantar/elonwallet-function/models"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"sync"
)

const (
	RegistrationKey  = "registration"
	LoginKey         = "login"
	TransactionKey   = "transaction"
	AddCredentialKey = "add_credential"
)

type Datastore interface {
	GetUser() (models.User, error)
	SaveUser(u models.User) error
	SaveSigningKey(s models.SigningKey) error
	GetSigningKey() (models.SigningKey, error)
}

type Api struct {
	w          *webauthn.WebAuthn
	d          Datastore
	mu         sync.Mutex
	signingKey models.SigningKey
}

func NewApi(d Datastore, signingKey models.SigningKey) (*Api, error) {
	w, err := webauthn.New(&webauthn.Config{
		RPID:          "localhost",
		RPDisplayName: "ElonWallet",
		RPOrigins:     []string{"http://localhost:3000"},
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			UserVerification: protocol.VerificationRequired,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create webauthn: %w", err)
	}

	return &Api{
		w:          w,
		d:          d,
		mu:         sync.Mutex{},
		signingKey: signingKey,
	}, nil
}
