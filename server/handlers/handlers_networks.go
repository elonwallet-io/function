package handlers

import (
	"fmt"
	"github.com/Leantar/elonwallet-function/models"
	"github.com/labstack/echo/v4"
	"net/http"
)

var networks = models.Networks{
	{
		Name:          "Ethereum Mainnet",
		Chain:         1,
		BlockExplorer: "https://etherscan.io/tx/",
		Currency:      "ETH",
		RPC:           "https://mainnet.infura.io/v3/a208d80edc8b44ea8b16e45171988562",
		Decimals:      18,
	},
	{
		Name:          "Goerli Testnet",
		Chain:         5,
		BlockExplorer: "https://goerli.etherscan.io/tx/",
		Currency:      "ETH",
		RPC:           "https://goerli.infura.io/v3/a208d80edc8b44ea8b16e45171988562",
		Decimals:      18,
	},
	{
		Name:          "Sepolia Testnet",
		Chain:         11155111,
		BlockExplorer: "https://sepolia.etherscan.io/tx/",
		Currency:      "ETH",
		RPC:           "https://sepolia.infura.io/v3/a208d80edc8b44ea8b16e45171988562",
		Decimals:      18,
	},
	{
		Name:          "Polygon Mainnet",
		Chain:         137,
		BlockExplorer: "https://polygonscan.com//tx/",
		Currency:      "MATIC",
		RPC:           "https://polygon-rpc.com/",
		Decimals:      18,
	},
	{
		Name:          "Mumbai Testnet",
		Chain:         80001,
		BlockExplorer: "https://mumbai.polygonscan.com/tx/",
		Currency:      "MATIC",
		RPC:           "https://rpc-mumbai.maticvigil.com/",
		Decimals:      18,
	},
}

func (a *Api) GetNetworks() echo.HandlerFunc {
	type networkWithChainHex struct {
		Name          string `json:"name"`
		Chain         string `json:"chain"`
		BlockExplorer string `json:"block_explorer"`
		Currency      string `json:"currency"`
		Decimals      int64  `json:"decimals"`
	}
	type output struct {
		Networks []networkWithChainHex `json:"networks"`
	}
	return func(c echo.Context) error {
		out := output{
			Networks: make([]networkWithChainHex, len(networks)),
		}
		i := 0
		for _, nw := range networks {
			out.Networks[i] = networkWithChainHex{
				Name:          nw.Name,
				Chain:         fmt.Sprintf("0x%x", nw.Chain),
				BlockExplorer: nw.BlockExplorer,
				Currency:      nw.Currency,
				Decimals:      nw.Decimals,
			}
			i++
		}

		return c.JSON(http.StatusOK, out)
	}
}
