package models

import (
	"fmt"
	"golang.org/x/exp/slices"
)

type Network struct {
	Name          string
	Chain         int64
	BlockExplorer string
	Currency      string
	RPC           string
	Decimals      int64
}

type Networks []Network

func (n Networks) FindByChain(chain string) (Network, bool) {
	index := slices.IndexFunc(n, func(network Network) bool {
		chainHex := fmt.Sprintf("0x%x", network.Chain)
		return chainHex == chain
	})
	if index == -1 {
		return Network{}, false
	}

	return n[index], true
}
