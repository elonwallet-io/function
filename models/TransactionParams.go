package models

import "github.com/ethereum/go-ethereum/core/types"

type TransactionParams struct {
	Type                 string            `json:"type"`
	Nonce                string            `json:"nonce"`
	To                   string            `json:"to"`
	From                 string            `json:"from"`
	Gas                  string            `json:"gas"`
	Value                string            `json:"value"`
	Input                string            `json:"input"`
	GasPrice             string            `json:"gas_price"`
	MaxPriorityFeePerGas string            `json:"max_priority_fee_per_gas"`
	MaxFeePerGas         string            `json:"max_fee_per_gas"`
	AccessList           *types.AccessList `json:"access_list"`
	ChainID              string            `json:"chain_id"`
}
