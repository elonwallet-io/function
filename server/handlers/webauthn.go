package handlers

import (
	"github.com/Leantar/elonwallet-function/models"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/protocol/webauthncose"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/labstack/echo/v4"
	"net/http"
)

func getCreationOptions(credentialExcludeList []protocol.CredentialDescriptor) webauthn.RegistrationOption {
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

func (a *Api) loginInitialize(user models.User, sessionKey string) (*protocol.CredentialAssertion, error) {
	options, session, err := a.w.BeginLogin(user.WebauthnData)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	user.WebauthnData.Sessions[sessionKey] = *session
	err = a.repo.UpsertUser(user)
	if err != nil {
		return nil, err
	}

	return options, nil
}

func (a *Api) loginFinalize(user models.User, sessionKey string, response protocol.CredentialAssertionResponse) (*webauthn.Credential, error) {
	session, ok := user.WebauthnData.Sessions[sessionKey]
	if !ok {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Transaction must be initialized beforehand")
	}
	delete(user.WebauthnData.Sessions, sessionKey)

	car, err := response.Parse()
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	cred, err := a.w.ValidateLogin(user.WebauthnData, session, car)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	err = a.repo.UpsertUser(user)
	if err != nil {
		return nil, err
	}

	return cred, nil
}
