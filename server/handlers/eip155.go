package handlers

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/Leantar/elonwallet-function/models"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"math/big"
	"net/http"
	"strconv"
)

var errInsufficientFunds = errors.New("insufficient funds for transaction")

type transactionParams struct {
	Chain    string `json:"chain" validate:"required,hexadecimal"`
	From     string `json:"from" validate:"required,eth_addr"`
	To       string `json:"to" validate:"required,eth_addr"`
	Data     string `json:"data" validate:"omitempty,hexadecimal"` //has currently no effect, because we only support the main chain
	Gas      string `json:"gas" validate:"omitempty,number"`       //optional as per WalletConnect definition
	GasPrice string `json:"gas_price" validate:"omitempty,number"` //optional as per WalletConnect definition
	Value    string `json:"value" validate:"omitempty,number"`     //optional as per WalletConnect definition
	Nonce    string `json:"nonce" validate:"omitempty,number"`     //optional as per WalletConnect definition
}

func createTransaction(params transactionParams, network models.Network, client *ethclient.Client, ctx context.Context) (*types.Transaction, error) {
	value, err := parseValue(params.Value)
	if err != nil {
		return nil, err
	}

	from := common.HexToAddress(params.From)
	to := common.HexToAddress(params.To)

	gas, err := parseGas(params.Gas)
	if err != nil {
		return nil, err
	}

	gasPrice, err := parseGasPrice(params.GasPrice, client, ctx)
	if err != nil {
		return nil, err
	}

	nonce, err := parseNonce(params.Nonce, from, client, ctx)
	if err != nil {
		return nil, err
	}

	tip, err := client.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to suggest tip cap: %w", err)
	}

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   new(big.Int).SetInt64(network.Chain),
		Nonce:     nonce,
		GasFeeCap: gasPrice,
		GasTipCap: tip,
		Gas:       gas,
		To:        &to,
		Value:     value,
		Data:      make([]byte, 0),
	})

	return tx, nil
}

func parseValue(value string) (*big.Int, error) {
	var parsed *big.Int
	if value != "" {
		var ok bool
		parsed, ok = new(big.Int).SetString(value, 10)
		if !ok {
			return nil, echo.NewHTTPError(http.StatusBadRequest, "value is not a valid number")
		}
	} else {
		parsed = new(big.Int).SetInt64(0)
	}

	return parsed, nil
}

func parseNonce(nonce string, from common.Address, client *ethclient.Client, ctx context.Context) (parsed uint64, err error) {
	if nonce != "" {
		parsed, err = strconv.ParseUint(nonce, 10, 64)
		if err != nil {
			return 0, echo.NewHTTPError(http.StatusBadRequest, "invalid nonce value").SetInternal(err)
		}
	} else {
		parsed, err = client.PendingNonceAt(ctx, from)
		if err != nil {
			return 0, fmt.Errorf("failed to get nonce: %w", err)
		}
	}

	return parsed, nil
}

func parseGasPrice(gasPrice string, client *ethclient.Client, ctx context.Context) (price *big.Int, err error) {
	if gasPrice != "" {
		var ok bool
		price, ok = new(big.Int).SetString(gasPrice, 10)
		if !ok {
			return nil, echo.NewHTTPError(http.StatusBadRequest, "gasPrice is not a valid number")
		}
	} else {
		price, err = client.SuggestGasPrice(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to suggest fee cap: %w", err)
		}
	}

	return price, nil
}

func parseGas(gas string) (parsed uint64, err error) {
	if gas != "" {
		parsed, err = strconv.ParseUint(gas, 10, 64)
		if err != nil {
			return 0, echo.NewHTTPError(http.StatusBadRequest, "invalid gas").SetInternal(err)
		}
	} else {
		parsed = 21000
	}

	return parsed, nil
}

func hasSufficientBalance(tx *types.Transaction, from common.Address, client *ethclient.Client, ctx context.Context) (bool, error) {
	balance, err := client.BalanceAt(ctx, from, nil)
	if err != nil {
		return false, fmt.Errorf("failed to get current balance: %w", err)
	}

	maxFee := new(big.Int).Mul(tx.GasFeeCap(), new(big.Int).SetUint64(tx.Gas()))
	total := new(big.Int).Add(tx.Value(), maxFee)

	if balance.Cmp(total) == -1 {
		return false, nil
	}

	return true, nil
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

func signTransaction(tx *types.Transaction, privateKeyHex string) (*types.Transaction, error) {
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		log.Fatal().Caller().Err(err).Msg("failed to convert hex to private key")
	}

	signedTx, err := types.SignTx(tx, types.LatestSignerForChainID(tx.ChainId()), privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign tx: %w", err)
	}

	return signedTx, nil
}

func createSignedTransaction(user models.User, params transactionParams, network models.Network, client *ethclient.Client, ctx context.Context) (*types.Transaction, error) {
	wallet, ok := user.Wallets.FindByAddress(params.From)
	if !ok {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "sending wallet does not exist")
	}

	tx, err := createTransaction(params, network, client, ctx)
	if err != nil {
		return nil, err
	}

	ok, err = hasSufficientBalance(tx, common.HexToAddress(wallet.Address), client, ctx)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, echo.NewHTTPError(http.StatusBadRequest, errInsufficientFunds.Error())
	}

	signedTx, err := signTransaction(tx, wallet.PrivateKeyHex)
	if err != nil {
		return nil, err
	}

	return signedTx, nil
}
