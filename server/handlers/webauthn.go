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

func (a *Api) loginInitialize(user *models.User, sessionKey string) (*protocol.CredentialAssertion, error) {
	options, session, err := a.w.BeginLogin(user.WebauthnData)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	removePendingTransaction(user, sessionKey)

	user.WebauthnData.Sessions[sessionKey] = *session
	return options, nil
}

func (a *Api) loginFinalize(user *models.User, req *http.Request, sessionKey string) (*webauthn.Credential, *webauthn.SessionData, error) {
	session, ok := user.WebauthnData.Sessions[sessionKey]
	if !ok {
		return nil, nil, echo.NewHTTPError(http.StatusBadRequest, "Please call the initialize endpoint first")
	}
	delete(user.WebauthnData.Sessions, sessionKey)

	cred, err := a.w.FinishLogin(user.WebauthnData, session, req)
	if err != nil {
		return nil, nil, echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return cred, &session, nil
}

func (a *Api) transactionInitialize(user *models.User, params *transactionParams, sessionKey string) (*protocol.CredentialAssertion, error) {
	options, err := a.loginInitialize(user, sessionKey)
	if err != nil {
		return nil, err
	}

	challenge := user.WebauthnData.Sessions[sessionKey].Challenge
	user.WebauthnData.PendingTransactions[challenge] = models.TransactionParams(*params)

	return options, nil
}

func (a *Api) transactionFinalize(user *models.User, req *http.Request, sessionKey string) (*transactionParams, error) {
	_, session, err := a.loginFinalize(user, req, sessionKey)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	params := user.WebauthnData.PendingTransactions[session.Challenge]
	delete(user.WebauthnData.PendingTransactions, session.Challenge)

	return (*transactionParams)(&params), nil
}

// Deletes a pending transaction corresponding to the sessionKey.
// Prevents bloating the disk if a user decides to not finish ongoing transactions before starting a new one
func removePendingTransaction(user *models.User, sessionKey string) {
	session, ok := user.WebauthnData.Sessions[sessionKey]
	if !ok {
		return
	}

	delete(user.WebauthnData.PendingTransactions, session.Challenge)
}
