package models

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/exp/slices"
)

type Wallet struct {
	Name          string `json:"name"`
	PrivateKeyHex string `json:"private_key_hex"`
	Address       string `json:"address"`
	Public        bool   `json:"public"`
}

func NewWallet(name string, public bool) (Wallet, error) {
	sk, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		return Wallet{}, fmt.Errorf("failed to generate ecdsa secret key: %w", err)
	}

	pk := sk.Public()
	pkECDSA := pk.(*ecdsa.PublicKey)

	return Wallet{
		Name:          name,
		PrivateKeyHex: hex.EncodeToString(crypto.FromECDSA(sk)),
		Address:       crypto.PubkeyToAddress(*pkECDSA).Hex(),
		Public:        public,
	}, nil
}

type Wallets []Wallet

func (w Wallets) FindByAddress(address string) (Wallet, bool) {
	index := slices.IndexFunc(w, func(wallet Wallet) bool {
		return wallet.Address == address
	})
	if index == -1 {
		return Wallet{}, false
	}

	return w[index], true
}

func (w Wallets) Exists(name string) bool {
	return slices.ContainsFunc(w, func(wallet Wallet) bool {
		return wallet.Name == name
	})
}

func (w Wallets) PublicWallets() []Wallet {
	wallets := make([]Wallet, 0)
	for _, wallet := range w {
		if wallet.Public {
			wallets = append(wallets, wallet)
		}
	}
	return wallets
}
