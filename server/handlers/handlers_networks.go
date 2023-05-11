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
		ChainID:       1,
		ChainIDHex:    fmt.Sprintf("0x%x", 1),
		BlockExplorer: "https://etherscan.io",
		Currency:      "ETH",
		RPC:           "https://mainnet.infura.io/v3/a208d80edc8b44ea8b16e45171988562",
		Decimals:      18,
		Testnet:       false,
	},
	{
		Name:          "Goerli Testnet",
		ChainID:       5,
		ChainIDHex:    fmt.Sprintf("0x%x", 5),
		BlockExplorer: "https://goerli.etherscan.io",
		Currency:      "ETH",
		RPC:           "https://goerli.infura.io/v3/a208d80edc8b44ea8b16e45171988562",
		Decimals:      18,
		Testnet:       true,
	},
	{
		Name:          "Sepolia Testnet",
		ChainID:       11155111,
		ChainIDHex:    fmt.Sprintf("0x%x", 11155111),
		BlockExplorer: "https://sepolia.etherscan.io",
		Currency:      "ETH",
		RPC:           "https://sepolia.infura.io/v3/a208d80edc8b44ea8b16e45171988562",
		Decimals:      18,
		Testnet:       true,
	},
	{
		Name:          "Polygon Mainnet",
		ChainID:       137,
		ChainIDHex:    fmt.Sprintf("0x%x", 137),
		BlockExplorer: "https://polygonscan.com",
		Currency:      "MATIC",
		RPC:           "https://polygon-rpc.com/",
		Decimals:      18,
		Testnet:       false,
	},
	{
		Name:          "Mumbai Testnet",
		ChainID:       80001,
		ChainIDHex:    fmt.Sprintf("0x%x", 80001),
		BlockExplorer: "https://mumbai.polygonscan.com",
		Currency:      "MATIC",
		RPC:           "https://rpc-mumbai.maticvigil.com/",
		Decimals:      18,
		Testnet:       true,
	},
	{
		Name:          "Avalanche C-Chain",
		ChainID:       43114,
		ChainIDHex:    fmt.Sprintf("0x%x", 43114),
		BlockExplorer: "https://snowtrace.io",
		Currency:      "AVAX",
		RPC:           "https://api.avax.network/ext/bc/C/rpc",
		Decimals:      18,
		Testnet:       false,
	},
	{
		Name:          "Fantom Opera",
		ChainID:       250,
		ChainIDHex:    fmt.Sprintf("0x%x", 250),
		BlockExplorer: "https://ftmscan.com",
		Currency:      "FTM",
		RPC:           "https://rpc2.fantom.network",
		Decimals:      18,
		Testnet:       false,
	},
	{
		Name:          "Arbitrum One",
		ChainID:       42161,
		ChainIDHex:    fmt.Sprintf("0x%x", 42161),
		BlockExplorer: "https://arbiscan.io",
		Currency:      "ETH",
		RPC:           "https://arb1.arbitrum.io/rpc",
		Decimals:      18,
		Testnet:       false,
	},
	{
		Name:          "Binance Smart Chain Mainnet",
		ChainID:       56,
		ChainIDHex:    fmt.Sprintf("0x%x", 56),
		BlockExplorer: "https://bscscan.com",
		Currency:      "BNB",
		RPC:           "https://bsc-dataseed.binance.org",
		Decimals:      18,
		Testnet:       false,
	},
}

func (a *Api) GetNetworks() echo.HandlerFunc {
	type output struct {
		Networks []models.Network `json:"networks"`
	}
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, output{networks})
	}
}
