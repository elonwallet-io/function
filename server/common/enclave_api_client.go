package common

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"github.com/Leantar/elonwallet-function/models"
	"net/http"
)

type EnclaveApiClient struct {
	url string
}

func NewEnclaveApiClient(url string) EnclaveApiClient {
	return EnclaveApiClient{
		url: url,
	}
}

func (e *EnclaveApiClient) SendEmergencyAccessInvitation(waitingPeriodInDays uint64) error {
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

func (e *EnclaveApiClient) SendEmergencyAccessGrantRevocation() error {
	return nil
}

func (e *EnclaveApiClient) GetJWTVerificationKey() (ed25519.PublicKey, error) {
	res, err := http.Get(fmt.Sprintf("%s/jwt-verification-key", e.url))
	if err != nil {
		return nil, fmt.Errorf("failed to get verification key: %w", err)
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("received error status code: %d", res.StatusCode)
	}

	type input struct {
		VerificationKey []byte `json:"verification_key"`
	}

	var in input
	if err := json.NewDecoder(res.Body).Decode(&in); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return in.VerificationKey, nil
}
