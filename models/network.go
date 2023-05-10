package models

import (
	"golang.org/x/exp/slices"
)

type Network struct {
	Name          string `json:"name"`
	ChainID       int64  `json:"chain_id"`
	ChainIDHex    string `json:"chain_id_hex"`
	BlockExplorer string `json:"block_explorer"`
	Currency      string `json:"currency"`
	RPC           string `json:"-"`
	Decimals      int64  `json:"decimals"`
	Testnet       bool   `json:"testnet"`
}

type Networks []Network

func (n Networks) FindByChainIDHex(chainIDHex string) (Network, bool) {
	index := slices.IndexFunc(n, func(network Network) bool {
		return network.ChainIDHex == chainIDHex
	})
	if index == -1 {
		return Network{}, false
	}

	return n[index], true
}
