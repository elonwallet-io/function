package common

import (
	"crypto/ed25519"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

func CreateJWT(credential, aud string, sk ed25519.PrivateKey) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims{
		"exp":        now.Add(time.Hour * 24).Unix(),
		"nbf":        now.Unix(),
		"iat":        now.Unix(),
		"credential": credential,
		"aud":        aud,
	})

	return token.SignedString(sk)
}

func ValidateJWT(tokenString string, pk ed25519.PublicKey) (jwt.MapClaims, bool) {
	parser := jwt.NewParser(
		jwt.WithIssuedAt(),
		jwt.WithValidMethods([]string{jwt.SigningMethodEdDSA.Alg()}),
		jwt.WithAudience("enclave"),
	)

	token, err := parser.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return pk, nil
	})

	if err != nil {
		fmt.Println(err)
		return nil, false
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		fmt.Println("invalid claims")
		return nil, false
	}

	return claims, true
}
