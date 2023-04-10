package models

import "crypto/ed25519"

type SigningKey struct {
	PrivateKey ed25519.PrivateKey `json:"private_key"`
	PublicKey  ed25519.PublicKey  `json:"public_key"`
}
