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
	"regexp"
	"strings"
)

var (
	errInsufficientFunds = errors.New("insufficient funds for transaction")
	leadingZeroesRegex   = regexp.MustCompile("^0+")
)

type transactionParams struct {
	Type                 string            `json:"type" validate:"required,oneof=0x1 0x2"`
	Nonce                string            `json:"nonce" validate:"omitempty,hexadecimal"`
	To                   string            `json:"to" validate:"required,ethereum_address"`
	From                 string            `json:"from" validate:"required,ethereum_address"`
	Gas                  string            `json:"gas" validate:"omitempty,hexadecimal"`
	Value                string            `json:"value" validate:"omitempty,hexadecimal"`
	Input                string            `json:"input"`
	GasPrice             string            `json:"gasPrice" validate:"omitempty,hexadecimal"`
	MaxPriorityFeePerGas string            `json:"maxPriorityFeePerGas" validate:"omitempty,hexadecimal"`
	MaxFeePerGas         string            `json:"maxFeePerGas" validate:"omitempty,hexadecimal"`
	AccessList           *types.AccessList `json:"accessList"`
	ChainID              string            `json:"chainId" validate:"omitempty,hexadecimal"`
}

type parsedCommonParams struct {
	Nonce uint64
	To    common.Address
	From  common.Address
	Value *big.Int
	Data  []byte
}

func createTransaction(params *transactionParams, network models.Network, client *ethclient.Client, ctx context.Context) (*types.Transaction, error) {
	if params.Type == "0x1" {
		if params.AccessList != nil {
			return createLegacyAccessListTransaction(params, client, ctx)
		} else {
			return createLegacyTransaction(params, client, ctx)
		}
	} else {
		return createDynamicFeeTransaction(params, network, client, ctx)
	}
}

func createLegacyAccessListTransaction(params *transactionParams, client *ethclient.Client, ctx context.Context) (*types.Transaction, error) {
	commonParams, err := parseCommonParams(params, client, ctx)
	if err != nil {
		return nil, err
	}

	gasPrice, err := parseGasPrice(params.GasPrice, client, ctx)
	if err != nil {
		return nil, err
	}

	tx := &types.AccessListTx{
		Nonce:      commonParams.Nonce,
		GasPrice:   gasPrice,
		To:         &commonParams.To,
		Value:      commonParams.Value,
		Data:       commonParams.Data,
		AccessList: *params.AccessList,
	}

	gas, err := parseGas(params.Gas, commonParams.From, types.NewTx(tx), client, ctx)
	if err != nil {
		return nil, err
	}
	tx.Gas = gas

	log.Debug().Msgf("tx: %v", tx)

	return types.NewTx(tx), nil
}

func createLegacyTransaction(params *transactionParams, client *ethclient.Client, ctx context.Context) (*types.Transaction, error) {
	commonParams, err := parseCommonParams(params, client, ctx)
	if err != nil {
		return nil, err
	}

	gasPrice, err := parseGasPrice(params.GasPrice, client, ctx)
	if err != nil {
		return nil, err
	}

	tx := &types.LegacyTx{
		Nonce:    commonParams.Nonce,
		GasPrice: gasPrice,
		To:       &commonParams.To,
		Value:    commonParams.Value,
		Data:     commonParams.Data,
	}

	gas, err := parseGas(params.Gas, commonParams.From, types.NewTx(tx), client, ctx)
	if err != nil {
		return nil, err
	}
	tx.Gas = gas

	log.Debug().Msgf("tx: %v", tx)

	return types.NewTx(tx), nil
}

func createDynamicFeeTransaction(params *transactionParams, network models.Network, client *ethclient.Client, ctx context.Context) (*types.Transaction, error) {
	commonParams, err := parseCommonParams(params, client, ctx)
	if err != nil {
		return nil, err
	}

	feeCap, err := parseFeeCap(params.GasPrice, client, ctx)
	if err != nil {
		return nil, err
	}

	tipCap, err := parseTipCap(params.MaxPriorityFeePerGas, client, ctx)
	if err != nil {
		return nil, err
	}

	tx := &types.DynamicFeeTx{
		ChainID:   new(big.Int).SetInt64(network.ChainID),
		Nonce:     commonParams.Nonce,
		GasFeeCap: feeCap,
		GasTipCap: tipCap,
		To:        &commonParams.To,
		Value:     commonParams.Value,
		Data:      commonParams.Data,
	}

	if params.AccessList != nil {
		tx.AccessList = *params.AccessList
	}

	gas, err := parseGas(params.Gas, commonParams.From, types.NewTx(tx), client, ctx)
	if err != nil {
		return nil, err
	}
	tx.Gas = gas

	log.Debug().Msgf("tx: %v", tx)

	return types.NewTx(tx), nil
}

func parseCommonParams(params *transactionParams, client *ethclient.Client, ctx context.Context) (*parsedCommonParams, error) {
	value, err := parseValue(params.Value)
	if err != nil {
		return nil, err
	}

	from := common.HexToAddress(params.From)
	to := common.HexToAddress(params.To)

	nonce, err := parseNonce(params.Nonce, from, client, ctx)
	if err != nil {
		return nil, err
	}

	data, err := parseData(params.Input)
	if err != nil {
		return nil, err
	}

	return &parsedCommonParams{
		Nonce: nonce,
		To:    to,
		From:  from,
		Value: value,
		Data:  data,
	}, nil
}

func parseValue(value string) (parsed *big.Int, err error) {
	if value != "" {
		value = replaceLeadingZeroesFromHexNumber(value)
		parsed, err = hexutil.DecodeBig(value)
		if err != nil {
			err = echo.NewHTTPError(http.StatusBadRequest, "Value is invalid").SetInternal(err)
		}
	} else {
		parsed = new(big.Int).SetInt64(0)
	}

	return
}

func parseData(data string) (parsed []byte, err error) {
	if len(data) == 0 {
		return make([]byte, 0), nil
	}

	parsed, err = hexutil.Decode(data)
	if err != nil {
		err = fmt.Errorf("failed to decode data hex string: %w", err)
	}

	return
}

func parseNonce(nonce string, from common.Address, client *ethclient.Client, ctx context.Context) (parsed uint64, err error) {
	if nonce != "" {
		nonce = replaceLeadingZeroesFromHexNumber(nonce)
		parsed, err = hexutil.DecodeUint64(nonce)
		if err != nil {
			err = echo.NewHTTPError(http.StatusBadRequest, "Nonce is invalid").SetInternal(err)
		}
	} else {
		parsed, err = client.PendingNonceAt(ctx, from)
		if err != nil {
			err = fmt.Errorf("failed to get nonce: %w", err)
		}
	}

	return
}

func parseGasPrice(gasPrice string, client *ethclient.Client, ctx context.Context) (parsed *big.Int, err error) {
	if gasPrice != "" {
		gasPrice = replaceLeadingZeroesFromHexNumber(gasPrice)
		parsed, err = hexutil.DecodeBig(gasPrice)
		if err != nil {
			err = echo.NewHTTPError(http.StatusBadRequest, "GasPrice is invalid").SetInternal(err)
		}
	} else {
		parsed, err = client.SuggestGasPrice(ctx)
		if err != nil {
			err = fmt.Errorf("failed to suggest gas price: %w", err)
		}
	}

	return
}

func parseTipCap(tipCap string, client *ethclient.Client, ctx context.Context) (parsed *big.Int, err error) {
	if tipCap != "" {
		tipCap = replaceLeadingZeroesFromHexNumber(tipCap)
		parsed, err = hexutil.DecodeBig(tipCap)
		if err != nil {
			err = echo.NewHTTPError(http.StatusBadRequest, "MaxPriorityFeePerGas is invalid").SetInternal(err)
		}
	} else {
		parsed, err = client.SuggestGasTipCap(ctx)
		if err != nil {
			err = fmt.Errorf("failed to suggest tipCap: %w", err)
		}
	}

	return
}

func parseFeeCap(feeCap string, client *ethclient.Client, ctx context.Context) (parsed *big.Int, err error) {
	if feeCap != "" {
		feeCap = replaceLeadingZeroesFromHexNumber(feeCap)
		parsed, err = hexutil.DecodeBig(feeCap)
		if err != nil {
			err = echo.NewHTTPError(http.StatusBadRequest, "MaxFeePerGas is invalid").SetInternal(err)
		}
	} else {
		parsed, err = client.SuggestGasPrice(ctx)
		if err != nil {
			err = fmt.Errorf("failed to suggest feeCap: %w", err)
		}
	}

	return
}

func parseGas(gas string, from common.Address, tx *types.Transaction, client *ethclient.Client, ctx context.Context) (parsed uint64, err error) {
	if gas != "" {
		gas = replaceLeadingZeroesFromHexNumber(gas)
		parsed, err = hexutil.DecodeUint64(gas)
		if err != nil {
			err = echo.NewHTTPError(http.StatusBadRequest, "Gas is invalid").SetInternal(err)
		}
	} else {
		parsed, err = client.EstimateGas(ctx, ethereum.CallMsg{
			From:       from,
			To:         tx.To(),
			GasFeeCap:  tx.GasFeeCap(),
			GasTipCap:  tx.GasTipCap(),
			GasPrice:   tx.GasPrice(),
			Value:      tx.Value(),
			Data:       tx.Data(),
			AccessList: tx.AccessList(),
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

	if tx.Type() == types.LegacyTxType || tx.Type() == types.AccessListTxType {
		maxFee := new(big.Int).Mul(tx.GasPrice(), new(big.Int).SetUint64(tx.Gas()))
		total := new(big.Int).Add(tx.Value(), maxFee)

		if balance.Cmp(total) == -1 {
			return false, nil
		}
	} else if tx.Type() == types.DynamicFeeTxType {
		maxFee := new(big.Int).Mul(tx.GasFeeCap(), new(big.Int).SetUint64(tx.Gas()))
		total := new(big.Int).Add(tx.Value(), maxFee)

		if balance.Cmp(total) == -1 {
			return false, nil
		}
	}

	return true, nil
}

// Must be done to prevent hexutil.DecodeBig and hexutil.DecodeUint64 error
func replaceLeadingZeroesFromHexNumber(hexNumber string) string {
	if !strings.HasPrefix(hexNumber, "0x") {
		return hexNumber
	}

	adjustedNumber := leadingZeroesRegex.ReplaceAllString(hexNumber[2:], "")

	if adjustedNumber == "" {
		adjustedNumber = "0"
	}
	return fmt.Sprintf("0x%s", adjustedNumber)
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

func createSignedTransaction(user models.User, params *transactionParams, network models.Network, client *ethclient.Client, ctx context.Context) (*types.Transaction, error) {
	wallet, ok := user.Wallets.FindByAddress(params.From)
	if !ok {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Sending wallet does not exist")
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
