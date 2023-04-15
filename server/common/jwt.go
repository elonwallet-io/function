package common

import (
	"crypto/ed25519"
	"errors"
	"github.com/Leantar/elonwallet-function/models"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

const (
	Enclave = "elonwallet-enclave"
	Backend = "elonwallet-backend"
)

type BackendClaims struct {
	jwt.RegisteredClaims
}

type EnclaveClaims struct {
	Credential string `json:"credential"`
	jwt.RegisteredClaims
}

func CreateBackendJWT(user models.User, sk ed25519.PrivateKey) (string, error) {
	now := time.Now()
	claims := BackendClaims{
		jwt.RegisteredClaims{
			Issuer:    Enclave,
			Subject:   user.Email,
			Audience:  []string{Backend},
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour * 24)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)

	return token.SignedString(sk)
}

func CreateEnclaveJWT(user models.User, currentCredential string, sk ed25519.PrivateKey) (string, error) {
	now := time.Now()
	claims := EnclaveClaims{
		currentCredential,
		jwt.RegisteredClaims{
			Issuer:    Enclave,
			Subject:   user.Email,
			Audience:  []string{Enclave},
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour * 24)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)

	return token.SignedString(sk)
}

func ValidateEnclaveJWT(tokenString, subject string, pk ed25519.PublicKey) (EnclaveClaims, error) {
	parser := jwt.NewParser(
		jwt.WithIssuedAt(),
		jwt.WithValidMethods([]string{jwt.SigningMethodEdDSA.Alg()}),
		jwt.WithAudience(Enclave),
		jwt.WithIssuer(Enclave),
		jwt.WithSubject(subject),
	)

	var claims EnclaveClaims
	_, err := parser.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
		return pk, nil
	})
	if err != nil {
		return EnclaveClaims{}, err
	}

	if claims.Credential == "" {
		return EnclaveClaims{}, errors.New("credential claim is missing")
	}

	return claims, nil
}
