package common

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"github.com/Leantar/elonwallet-function/models"
	"net/http"
)

type BackendApiClient struct {
	url string
	jwt string
}

func NewBackendApiClient(url string, user models.User, sk ed25519.PrivateKey) (BackendApiClient, error) {
	jwt, err := CreateBackendJWT(user, "enclave", sk)
	if err != nil {
		return BackendApiClient{}, fmt.Errorf("failed to create jwt: %w", err)
	}

	return BackendApiClient{
		url: url,
		jwt: jwt,
	}, nil
}

func (b *BackendApiClient) PublishWalletInitialize(wallet models.Wallet) (challenge string, err error) {
	url := fmt.Sprintf("%s/users/my/wallets/initialize", b.url)
	type payload struct {
		Address string `json:"address"`
	}

	type response struct {
		Challenge string `json:"challenge"`
	}

	var out response
	err = b.doPostRequest(url, payload{wallet.Address}, &out)
	if err != nil {
		return "", err
	}

	return out.Challenge, nil
}

func (b *BackendApiClient) PublishWalletFinalize(wallet models.Wallet, signature string) error {
	url := fmt.Sprintf("%s/users/my/wallets/finalize", b.url)
	type payload struct {
		Name      string `json:"name"`
		Address   string `json:"address"`
		Signature string `json:"signature"`
	}

	p := payload{
		Name:      wallet.Name,
		Address:   wallet.Address,
		Signature: signature,
	}

	return b.doPostRequest(url, p, nil)
}

func (b *BackendApiClient) doPostRequest(url string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal json: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to instantiate request: %w", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", b.jwt))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != 200 && res.StatusCode != 201 {
		return fmt.Errorf("received error status code: %d", res.StatusCode)
	}

	if out != nil {
		if err := json.NewDecoder(res.Body).Decode(out); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}
