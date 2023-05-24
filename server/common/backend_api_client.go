package common

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"github.com/Leantar/elonwallet-function/models"
	"net/http"
	"net/url"
)

type BackendApiClient struct {
	url string
	jwt string
}

func NewBackendApiClient(url string, user models.User, sk ed25519.PrivateKey) (BackendApiClient, error) {
	var jwt string
	if sk != nil {
		var err error
		jwt, err = CreateBackendJWT(user, ScopeEnclave, sk)
		if err != nil {
			return BackendApiClient{}, fmt.Errorf("failed to create jwt: %w", err)
		}
	}

	return BackendApiClient{
		url: url,
		jwt: jwt,
	}, nil
}

func (b *BackendApiClient) PublishWalletInitialize(wallet models.Wallet) (challenge string, err error) {
	backendURL := fmt.Sprintf("%s/users/my/wallets/initialize", b.url)
	type payload struct {
		Address string `json:"address"`
	}

	type response struct {
		Challenge string `json:"challenge"`
	}

	var out response
	err = doPostRequestWithBearer(backendURL, payload{wallet.Address}, &out, b.jwt)
	if err != nil {
		return "", err
	}

	return out.Challenge, nil
}

func (b *BackendApiClient) PublishWalletFinalize(wallet models.Wallet, signature string) error {
	backendURL := fmt.Sprintf("%s/users/my/wallets/finalize", b.url)
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

	return doPostRequestWithBearer(backendURL, p, nil, b.jwt)
}

func (b *BackendApiClient) GetEnclaveURL(email string) (string, error) {
	escapedEmail := url.QueryEscape(email)
	res, err := http.Get(fmt.Sprintf("%s/users/%s/enclave-url?questioner=enclave", b.url, escapedEmail))
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		return "", fmt.Errorf("received error status code: %d", res.StatusCode)
	}

	type input struct {
		EnclaveURL string `json:"enclave_url"`
	}

	var in input
	if err := json.NewDecoder(res.Body).Decode(&in); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}
	return in.EnclaveURL, nil
}

func (b *BackendApiClient) SendNotification(title, body string) error {
	backendURL := fmt.Sprintf("%s/notifications", b.url)
	type payload struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}

	p := payload{
		Title: title,
		Body:  body,
	}

	return doPostRequestWithBearer(backendURL, p, nil, b.jwt)
}

func (b *BackendApiClient) ScheduleNotificationSeries(notifications []models.ScheduledNotification) (string, error) {
	backendURL := fmt.Sprintf("%s/notifications/series", b.url)
	type payload struct {
		Notifications []models.ScheduledNotification `json:"notifications"`
	}

	p := payload{
		Notifications: notifications,
	}

	type response struct {
		SeriesID string `json:"series_id"`
	}

	var out response
	err := doPostRequestWithBearer(backendURL, p, &out, b.jwt)
	if err != nil {
		return "", err
	}

	return out.SeriesID, nil
}

func (b *BackendApiClient) DeleteNotificationSeries(seriesID string) error {
	backendURL := fmt.Sprintf("%s/notifications/series/%s", b.url, seriesID)

	req, err := http.NewRequest(http.MethodDelete, backendURL, nil)
	if err != nil {
		return fmt.Errorf("failed to instantiate request: %w", err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", b.jwt))

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

func (b *BackendApiClient) DeleteUser(authorizationJWT string) error {
	backendURL := fmt.Sprintf("%s/users", b.url)

	req, err := http.NewRequest(http.MethodDelete, backendURL, nil)
	if err != nil {
		return fmt.Errorf("failed to instantiate request: %w", err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", authorizationJWT))

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
