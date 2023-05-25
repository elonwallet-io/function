package handlers

import (
	"fmt"
	"github.com/Leantar/elonwallet-function/models"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/labstack/echo/v4"
	"net/http"
)

func (a *Api) HandleSendTransactionInitialize() echo.HandlerFunc {
	return func(c echo.Context) error {
		user := c.Get("user").(models.User)

		options, err := a.loginInitialize(user, SendTransactionKey)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, options)
	}
}

func (a *Api) HandleSendTransactionFinalize() echo.HandlerFunc {
	type input struct {
		AssertionResponse protocol.CredentialAssertionResponse `json:"assertion_response"`
		TransactionParams transactionParams                    `json:"transaction_params"`
	}

	type output struct {
		Hash string `json:"hash"`
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

		_, err := a.loginFinalize(user, SendTransactionKey, in.AssertionResponse)
		if err != nil {
			return err
		}

		network, ok := networks.FindByChainIDHex(in.TransactionParams.Chain)
		if !ok {
			return echo.NewHTTPError(http.StatusBadRequest, "Network does not exist")
		}

		client, err := ethclient.DialContext(c.Request().Context(), network.RPC)
		if err != nil {
			return fmt.Errorf("failed to dial rpc: %w", err)
		}

		signedTx, err := createSignedTransaction(user, in.TransactionParams, network, client, c.Request().Context())
		if err != nil {
			return err
		}

		err = client.SendTransaction(c.Request().Context(), signedTx)
		if err != nil {
			return fmt.Errorf("failed to send tx: %w", err)
		}

		return c.JSON(http.StatusOK, output{signedTx.Hash().Hex()})
	}
}

func (a *Api) HandleSignTransactionInitialize() echo.HandlerFunc {
	return func(c echo.Context) error {
		user := c.Get("user").(models.User)

		options, err := a.loginInitialize(user, SignTransactionKey)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, options)
	}
}

func (a *Api) HandleSignTransactionFinalize() echo.HandlerFunc {
	type input struct {
		AssertionResponse protocol.CredentialAssertionResponse `json:"assertion_response"`
		TransactionParams transactionParams                    `json:"transaction_params"`
	}

	type output struct {
		Transaction string `json:"transaction"`
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

		_, err := a.loginFinalize(user, SignTransactionKey, in.AssertionResponse)
		if err != nil {
			return err
		}

		network, ok := networks.FindByChainIDHex(in.TransactionParams.Chain)
		if !ok {
			return echo.NewHTTPError(http.StatusBadRequest, "Network does not exist")
		}

		client, err := ethclient.DialContext(c.Request().Context(), network.RPC)
		if err != nil {
			return fmt.Errorf("failed to dial rpc: %w", err)
		}

		signedTx, err := createSignedTransaction(user, in.TransactionParams, network, client, c.Request().Context())
		if err != nil {
			return err
		}

		txBytes, err := signedTx.MarshalBinary()
		if err != nil {
			return fmt.Errorf("failed to marshal signed tx")
		}

		return c.JSON(http.StatusOK, output{hexutil.Encode(txBytes)})
	}
}
