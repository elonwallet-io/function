package common

import (
	"crypto/ed25519"
	"github.com/Leantar/elonwallet-function/models"
	"strings"
)

type EnclaveApiClient struct {
	url string
}

func NewEnclaveApiClient(url string, developmentMode bool) EnclaveApiClient {
	if developmentMode {
		url = strings.Replace(url, "localhost", "host.docker.internal", 1)
	}

	return EnclaveApiClient{
		url: url,
	}
}

func (e *EnclaveApiClient) SendEmergencyAccessInvitation(invitation models.EmergencyAccessInvitation) error {
	return nil
}

func (e *EnclaveApiClient) SendEmergencyAccessRequest() error {
	return nil
}

func (e *EnclaveApiClient) SendEmergencyAccessTakeoverRequest() ([]models.Wallet, error) {
	return nil, nil
}

func (e *EnclaveApiClient) SendEmergencyAccessInvitationResponse(accept bool) error {
	return nil
}

func (e *EnclaveApiClient) GetJWTVerificationKey() (ed25519.PublicKey, error) {
	return nil, nil
}
