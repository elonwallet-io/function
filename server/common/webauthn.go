package common

import (
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/protocol/webauthncose"
	"github.com/go-webauthn/webauthn/webauthn"
)

func GetCreationOptions(credentialExcludeList []protocol.CredentialDescriptor) webauthn.RegistrationOption {
	return func(creationOptions *protocol.PublicKeyCredentialCreationOptions) {
		creationOptions.Parameters = []protocol.CredentialParameter{
			{
				Type:      protocol.PublicKeyCredentialType,
				Algorithm: webauthncose.AlgEdDSA,
			},
			{
				Type:      protocol.PublicKeyCredentialType,
				Algorithm: webauthncose.AlgES256,
			},
			{
				Type:      protocol.PublicKeyCredentialType,
				Algorithm: webauthncose.AlgRS256,
			},
		}
		creationOptions.Attestation = protocol.PreferNoAttestation
		creationOptions.AuthenticatorSelection.UserVerification = protocol.VerificationRequired
		creationOptions.CredentialExcludeList = credentialExcludeList
	}
}
