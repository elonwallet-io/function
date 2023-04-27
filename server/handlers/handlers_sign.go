package handlers

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/Leantar/elonwallet-function/models"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"net/http"
	"regexp"
)

const (
	isHexRegexString = "^(0[xX])?[0-9a-fA-F]+$"
)

var (
	isHexRegex = regexp.MustCompile(isHexRegexString)
)

func (a *Api) CreatePersonalSignature() echo.HandlerFunc {
	type input struct {
		Message string `json:"message" validate:"required"`
		Chain   string `json:"chain" validate:"required,hexadecimal"`
		From    string `json:"from" validate:"required,eth_addr"`
	}

	type output struct {
		Signature string `json:"signature"`
	}
	return func(c echo.Context) error {
		var in input
		if err := c.Bind(&in); err != nil {
			return err
		}
		if err := c.Validate(&in); err != nil {
			return err
		}

		user := c.Get("user").(models.User)

		wallet, ok := user.Wallets.FindByAddress(in.From)
		if !ok {
			return echo.NewHTTPError(http.StatusBadRequest, "signing wallet does not exist")
		}

		privateKey, err := crypto.HexToECDSA(wallet.PrivateKeyHex)
		if err != nil {
			log.Fatal().Caller().Err(err).Msg("failed to convert hex to private key")
		}

		if isHexString(in.Message) {
			fmt.Println("it is hex")
			msgBytes, err := hexutil.Decode(in.Message)
			if err != nil {
				return fmt.Errorf("failed to decode hex message: %w", err)
			}
			in.Message = string(msgBytes)
			fmt.Println(in.Message)
		}

		signature, err := signMessage(in.Message, privateKey)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, output{signature})

	}
}

func signMessage(message string, privateKey *ecdsa.PrivateKey) (string, error) {
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), message)
	digest := crypto.Keccak256Hash([]byte(msg))
	sig, err := crypto.Sign(digest.Bytes(), privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign message: %w", err)
	}
	// Last byte from go signature is the recovery id instead of v. It needs to be overridden
	// See https://stackoverflow.com/questions/69762108/implementing-ethereum-personal-sign-eip-191-from-go-ethereum-gives-different-s for more info
	sig[64] += 27
	return hexutil.Encode(sig), nil
}

func isHexString(message string) bool {
	return isHexRegex.MatchString(message)
}
