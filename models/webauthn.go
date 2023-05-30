package models

import (
	"encoding/base64"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

type WebauthnData struct {
	ID                  string                          `json:"id"`
	Name                string                          `json:"name"`
	DisplayName         string                          `json:"display_name"`
	Credentials         map[string]webauthn.Credential  `json:"credentials"`
	Sessions            map[string]webauthn.SessionData `json:"sessions"`
	PendingTransactions map[string]TransactionParams    `json:"pending_transactions"` //key is the challenge string
}

func (w WebauthnData) WebAuthnID() []byte {
	return []byte(w.ID)
}

func (w WebauthnData) WebAuthnName() string {
	return w.Name
}

func (w WebauthnData) WebAuthnDisplayName() string {
	return w.DisplayName
}

func (w WebauthnData) WebAuthnIcon() string {
	return ""
}

func (w WebauthnData) WebAuthnCredentials() []webauthn.Credential {
	credentials := make([]webauthn.Credential, len(w.Credentials))
	i := 0
	for _, cred := range w.Credentials {
		credentials[i] = cred
		i++
	}
	return credentials
}

func (w WebauthnData) CredentialExcludeList() []protocol.CredentialDescriptor {
	credentialExcludeList := make([]protocol.CredentialDescriptor, len(w.Credentials))
	i := 0
	for _, cred := range w.Credentials {
		encodedBuffer := make([]byte, base64.RawURLEncoding.EncodedLen(len(cred.ID)))
		base64.RawURLEncoding.Encode(encodedBuffer, cred.ID)

		descriptor := protocol.CredentialDescriptor{
			Type:         protocol.PublicKeyCredentialType,
			CredentialID: encodedBuffer,
		}
		credentialExcludeList[i] = descriptor
		i++
	}
	return credentialExcludeList
}
