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
	jwt string
}

func NewEnclaveApiClient(url string, user models.User, sk ed25519.PrivateKey) (EnclaveApiClient, error) {
	var jwt string
	if sk != nil {
		var err error
		jwt, err = CreateBackendJWT(user, ScopeEnclave, sk)
		if err != nil {
			return EnclaveApiClient{}, fmt.Errorf("failed to create jwt: %w", err)
		}
	}

	return EnclaveApiClient{
		url: url,
		jwt: jwt,
	}, nil
}

func (e *EnclaveApiClient) InviteEmergencyAccessContact() error {
	enclaveURL := fmt.Sprintf("%s/emergency-access/grants", e.url)

	req, err := http.NewRequest(http.MethodPost, enclaveURL, nil)
	if err != nil {
		return fmt.Errorf("failed to instantiate request: %w", err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", e.jwt))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode < 200 || res.StatusCode > 299 {
		return fmt.Errorf("received error status code: %d", res.StatusCode)
	}

	return nil
}

func (e *EnclaveApiClient) RequestEmergencyAccess() (int64, error) {
	enclaveURL := fmt.Sprintf("%s/emergency-access/contacts/request-access", e.url)
	res, err := http.Get(enclaveURL)
	if err != nil {
		return 0, fmt.Errorf("failed to make request: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		return 0, fmt.Errorf("received error status code: %d", res.StatusCode)
	}

	type response struct {
		TakeoverAllowedAfter int64 `json:"takeover_allowed_after"`
	}

	var resp response
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	return resp.TakeoverAllowedAfter, nil
}

func (e *EnclaveApiClient) RequestEmergencyAccessTakeover() ([]models.Wallet, error) {
	enclaveURL := fmt.Sprintf("%s/emergency-access/contacts/request-access", e.url)
	res, err := http.Get(enclaveURL)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, fmt.Errorf("received error status code: %d", res.StatusCode)
	}

	type response struct {
		Wallets []models.Wallet
	}

	var in response
	if err := json.NewDecoder(res.Body).Decode(&in); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return in.Wallets, nil
}

func (e *EnclaveApiClient) RespondEmergencyAccessInvitation(accept bool) error {
	enclaveURL := fmt.Sprintf("%s/emergency-access/contacts/grant-response", e.url)
	type payload struct {
		Accept bool `json:"accept"`
	}

	p := payload{
		accept,
	}

	return doPostRequestWithBearer(enclaveURL, p, nil, e.jwt)
}

func (e *EnclaveApiClient) RemoveEmergencyAccessGrant() error {
	enclaveURL := fmt.Sprintf("%s/emergency-access/grants", e.url)

	req, err := http.NewRequest(http.MethodDelete, enclaveURL, nil)
	if err != nil {
		return fmt.Errorf("failed to instantiate request: %w", err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", e.jwt))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode < 200 || res.StatusCode > 299 {
		return fmt.Errorf("received error status code: %d", res.StatusCode)
	}

	return nil
}

func (e *EnclaveApiClient) DenyEmergencyAccessRequest() error {
	enclaveURL := fmt.Sprintf("%s/emergency-access/grants/deny-access-request", e.url)

	req, err := http.NewRequest(http.MethodPost, enclaveURL, nil)
	if err != nil {
		return fmt.Errorf("failed to instantiate request: %w", err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", e.jwt))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode < 200 || res.StatusCode > 299 {
		return fmt.Errorf("received error status code: %d", res.StatusCode)
	}

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
