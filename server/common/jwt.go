package common

import (
	"crypto/ed25519"
	"errors"
	"github.com/Leantar/elonwallet-function/models"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

const (
	Enclave               = "elonwallet-enclave"
	Backend               = "elonwallet-backend"
	ScopeUser             = "user"
	ScopeEnclave          = "enclave"
	ScopeCreateCredential = "create-credential"
)

type BackendClaims struct {
	Scope string `json:"scope"`
	jwt.RegisteredClaims
}

type EnclaveClaims struct {
	Scope      string `json:"scope"`
	Credential string `json:"credential"`
	jwt.RegisteredClaims
}

func CreateBackendJWT(user models.User, scope string, sk ed25519.PrivateKey) (string, error) {
	now := time.Now()
	claims := BackendClaims{
		scope,
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

func CreateEnclaveJWT(user models.User, scope, credential string, sk ed25519.PrivateKey) (string, error) {
	now := time.Now()
	claims := EnclaveClaims{
		scope,
		credential,
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

func ValidateJWT(tokenString string, keyFunc jwt.Keyfunc) (EnclaveClaims, error) {
	parser := jwt.NewParser(
		jwt.WithIssuedAt(),
		jwt.WithValidMethods([]string{jwt.SigningMethodEdDSA.Alg()}),
		jwt.WithAudience(Enclave),
		jwt.WithIssuer(Enclave),
	)

	var claims EnclaveClaims
	_, err := parser.ParseWithClaims(tokenString, &claims, keyFunc)
	if err != nil {
		return EnclaveClaims{}, err
	}

	if claims.Scope == "" {
		return EnclaveClaims{}, errors.New("scope is missing")
	}

	return claims, nil
}
