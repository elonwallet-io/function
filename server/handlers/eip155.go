package handlers

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/Leantar/elonwallet-function/models"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"math/big"
	"net/http"
	"strconv"
)

var errInsufficientFunds = errors.New("insufficient funds for transaction")

type transactionParams struct {
	Chain    string `json:"chain" validate:"required,hexadecimal"`
	From     string `json:"from" validate:"required,ethereum_address"`
	To       string `json:"to" validate:"required,ethereum_address"`
	Data     string `json:"data"`
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

	feeCap, err := parseFeeCap(params.GasPrice, client, ctx)
	if err != nil {
		return nil, err
	}

	nonce, err := parseNonce(params.Nonce, from, client, ctx)
	if err != nil {
		return nil, err
	}

	tipCap, err := client.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to suggest tipCap cap: %w", err)
	}

	data, err := parseData(params.Data)
	if err != nil {
		return nil, err
	}

	dynTx := &types.DynamicFeeTx{
		ChainID:   new(big.Int).SetInt64(network.Chain),
		Nonce:     nonce,
		GasFeeCap: feeCap,
		GasTipCap: tipCap,
		To:        &to,
		Value:     value,
		Data:      data,
	}

	gas, err := parseGas(params.Gas, from, dynTx, client, ctx)
	if err != nil {
		return nil, err
	}
	dynTx.Gas = gas

	log.Debug().Msgf("tx: %v", dynTx)

	tx := types.NewTx(dynTx)
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

func parseData(hexData string) ([]byte, error) {
	if len(hexData) == 0 {
		return make([]byte, 0), nil
	}

	data, err := hexutil.Decode(hexData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode data hex string: %w", err)
	}

	return data, nil
}

func parseNonce(nonce string, from common.Address, client *ethclient.Client, ctx context.Context) (parsed uint64, err error) {
	if nonce != "" && nonce != "0" {
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

func parseFeeCap(feeCap string, client *ethclient.Client, ctx context.Context) (price *big.Int, err error) {
	if feeCap != "" {
		var ok bool
		price, ok = new(big.Int).SetString(feeCap, 10)
		if !ok {
			return nil, echo.NewHTTPError(http.StatusBadRequest, "feeCap is not a valid number")
		}
	} else {
		price, err = client.SuggestGasPrice(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to suggest fee cap: %w", err)
		}
	}

	return price, nil
}

func parseGas(gas string, from common.Address, dynTx *types.DynamicFeeTx, client *ethclient.Client, ctx context.Context) (parsed uint64, err error) {
	if gas != "" {
		parsed, err = strconv.ParseUint(gas, 10, 64)
		if err != nil {
			return 0, echo.NewHTTPError(http.StatusBadRequest, "invalid gas").SetInternal(err)
		}
	} else {
		parsed, err = client.EstimateGas(ctx, ethereum.CallMsg{
			From:      from,
			To:        dynTx.To,
			GasFeeCap: dynTx.GasFeeCap,
			GasTipCap: dynTx.GasTipCap,
			Value:     dynTx.Value,
			Data:      dynTx.Data,
		})
		if err != nil {
			return 0, fmt.Errorf("failed to estimate gas: %w", err)
		}
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

// EIP191 https://eips.ethereum.org/EIPS/eip-191
func signPersonal(message string, privateKey *ecdsa.PrivateKey) (string, error) {
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), message)
	hash := crypto.Keccak256Hash([]byte(msg))

	return signEncodedMessage(hash, privateKey)
}

// EIP712 https://eips.ethereum.org/EIPS/eip-712
func signTypedData(data apitypes.TypedData, privateKey *ecdsa.PrivateKey) (string, error) {
	domainSeparator, err := data.HashStruct("EIP712Domain", data.Domain.Map())
	if err != nil {
		return "", fmt.Errorf("failed to hash domainSeparator: %w", err)
	}

	message, err := data.HashStruct(data.PrimaryType, data.Message)
	if err != nil {
		return "common.Hash{}", fmt.Errorf("failed to hash message: %w", err)
	}

	msg := fmt.Sprintf("\x19\x01%s%s", string(domainSeparator), string(message))
	hash := crypto.Keccak256Hash([]byte(msg))

	return signEncodedMessage(hash, privateKey)
}

func signEncodedMessage(hash common.Hash, privateKey *ecdsa.PrivateKey) (string, error) {
	sig, err := crypto.Sign(hash.Bytes(), privateKey)
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
