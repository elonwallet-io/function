package handlers

import (
	"context"
	"errors"
	"fmt"
	"github.com/Leantar/elonwallet-function/models"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"math/big"
	"net/http"
)

var ErrInsufficientFunds = errors.New("insufficient funds for transaction")

type transactionData struct {
	wallet  models.Wallet
	network models.Network
	to      common.Address
	amount  *big.Int
}

func (a *Api) TransactionInitialize() echo.HandlerFunc {
	return func(c echo.Context) error {
		user := c.Get("user").(models.User)

		options, session, err := a.w.BeginLogin(user.WebauthnData)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		user.WebauthnData.Sessions[TransactionKey] = *session

		err = a.repo.UpsertUser(user)
		if err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}

		return c.JSON(http.StatusOK, options)
	}
}

func (a *Api) TransactionFinalize() echo.HandlerFunc {
	type transactionInfo struct {
		Chain  string `json:"chain" validate:"required,hexadecimal"`
		From   string `json:"from" validate:"required,ethereum_address"`
		To     string `json:"to" validate:"required,ethereum_address"`
		Amount string `json:"amount" validate:"required,number"`
	}

	type input struct {
		AssertionResponse protocol.CredentialAssertionResponse `json:"assertion_response"`
		TransactionInfo   transactionInfo                      `json:"transaction_info"`
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

		session, ok := user.WebauthnData.Sessions[TransactionKey]
		if !ok {
			return echo.NewHTTPError(http.StatusBadRequest, "transaction must be initialized beforehand")
		}
		delete(user.WebauthnData.Sessions, TransactionKey)

		car, err := in.AssertionResponse.Parse()
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		_, err = a.w.ValidateLogin(user.WebauthnData, session, car)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		wallet, ok := user.Wallets.FindByAddress(in.TransactionInfo.From)
		if !ok {
			return echo.NewHTTPError(http.StatusBadRequest, "sending wallet does not exist")
		}

		network, ok := networks.FindByChain(in.TransactionInfo.Chain)
		if !ok {
			return echo.NewHTTPError(http.StatusBadRequest, "network does not exist")
		}

		amount, ok := new(big.Int).SetString(in.TransactionInfo.Amount, 10)
		if !ok {
			return echo.NewHTTPError(http.StatusBadRequest, "amount is not a valid number")
		}

		td := transactionData{
			wallet:  wallet,
			network: network,
			to:      common.HexToAddress(in.TransactionInfo.To),
			amount:  amount,
		}

		if err := sendNativeTransaction(td, c.Request().Context()); err != nil {
			if err == ErrInsufficientFunds {
				return echo.NewHTTPError(http.StatusBadRequest, "You have insufficient funds for this transaction")
			}
			return fmt.Errorf("failed to send transaction: %w", err)
		}

		err = a.repo.UpsertUser(user)
		if err != nil {
			return err
		}

		return c.NoContent(http.StatusOK)
	}
}

func sendNativeTransaction(data transactionData, ctx context.Context) error {
	privateKey, err := crypto.HexToECDSA(data.wallet.PrivateKeyHex)
	if err != nil {
		log.Fatal().Caller().Err(err).Msg("failed to convert hex to private key")
	}

	client, err := ethclient.DialContext(ctx, data.network.RPC)
	if err != nil {
		return fmt.Errorf("failed to dial rpc: %w", err)
	}

	fromAddress := common.HexToAddress(data.wallet.Address)

	nonce, err := client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return fmt.Errorf("failed to get nonce: %w", err)
	}

	feeCap, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("failed to suggest fee cap: %w", err)
	}

	//Check balance to prevent error later
	balance, err := client.BalanceAt(ctx, fromAddress, nil)
	if err != nil {
		return fmt.Errorf("failed to get current balance: %w", err)
	}

	//Check if user has enough balance
	if balance.Cmp(new(big.Int).Add(data.amount, feeCap)) == -1 {
		return ErrInsufficientFunds
	}

	tip, err := client.SuggestGasTipCap(ctx)
	if err != nil {
		return fmt.Errorf("failed to suggest tip cap: %w", err)
	}

	chainID := new(big.Int).SetInt64(data.network.Chain)

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasFeeCap: feeCap,
		GasTipCap: tip,
		Gas:       21000,
		To:        &data.to,
		Value:     data.amount,
		Data:      make([]byte, 0),
	})

	signedTx, err := types.SignTx(tx, types.LatestSignerForChainID(chainID), privateKey)
	if err != nil {
		return fmt.Errorf("failed to sign tx: %w", err)
	}

	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		return fmt.Errorf("failed to send tx: %w", err)
	}

	log.Info().Caller().Msgf("sending transaction with hash: %s", signedTx.Hash().Hex())
	return nil
}
