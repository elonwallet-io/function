package handlers

import (
	"fmt"
	"github.com/Leantar/elonwallet-function/models"
	"github.com/Leantar/elonwallet-function/server/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/rs/zerolog/log"
)

func (a *Api) createWallet(name string, public bool, user models.User) (models.Wallet, error) {
	wallet, err := models.NewWallet(name, public)
	if err != nil {
		return models.Wallet{}, fmt.Errorf("failed to create new wallet: %w", err)
	}

	if public {
		b, err := common.NewBackendApiClient(a.cfg.BackendURL, user, a.signingKey.PrivateKey)
		if err != nil {
			return models.Wallet{}, fmt.Errorf("failed to create backend api client: %w", err)
		}

		challenge, err := b.PublishWalletInitialize(wallet)
		if err != nil {
			return models.Wallet{}, fmt.Errorf("failed to initalize wallet publication: %w", err)
		}

		privateKey, err := crypto.HexToECDSA(wallet.PrivateKeyHex)
		if err != nil {
			log.Fatal().Caller().Err(err).Msg("failed to convert hex to private key")
		}

		signature, err := signPersonal(challenge, privateKey)
		if err != nil {
			return models.Wallet{}, fmt.Errorf("failed to sign wallet challenge: %w", err)
		}

		err = b.PublishWalletFinalize(wallet, signature)
		if err != nil {
			return models.Wallet{}, fmt.Errorf("failed to finalize wallet publication: %w", err)
		}
	}

	return wallet, nil
}
