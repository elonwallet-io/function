package handlers

import (
	"github.com/Leantar/elonwallet-function/models"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"net/http"
)

func (a *Api) HandleSignPersonal() echo.HandlerFunc {
	type input struct {
		Message string `json:"message" validate:"required"`
		From    string `json:"from" validate:"required,ethereum_address"`
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

		signature, err := signPersonal(in.Message, privateKey)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, output{signature})

	}
}

func (a *Api) HandleSignTypedData() echo.HandlerFunc {
	type input struct {
		Data apitypes.TypedData `json:"typed_data" validate:"required"`
		From string             `json:"from" validate:"required,ethereum_address"`
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

		signature, err := signTypedData(in.Data, privateKey)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, output{signature})
	}
}
