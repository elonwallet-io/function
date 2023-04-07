package server

import (
	"fmt"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/labstack/echo/v4"
	"math/big"
	"net/http"
)

func (a *Api) EstimateFees() echo.HandlerFunc {
	type input struct {
		Chain string `validate:"required,hexadecimal"`
	}
	type output struct {
		EstimatedFees string `json:"estimated_fees"`
		BaseFee       string `json:"base_fee"`
		Tip           string `json:"tip"`
	}
	return func(c echo.Context) error {
		in := input{
			Chain: c.QueryParam("chain"),
		}
		if err := c.Validate(&in); err != nil {
			return err
		}

		network, ok := networks.FindByChain(in.Chain)
		if !ok {
			return echo.NewHTTPError(http.StatusBadRequest, "network does not exist")
		}

		client, err := ethclient.DialContext(c.Request().Context(), network.RPC)
		if err != nil {
			return fmt.Errorf("failed to dial rpc: %w", err)
		}

		fees, err := client.SuggestGasPrice(c.Request().Context())
		if err != nil {
			return fmt.Errorf("failed to suggest fees: %w", err)
		}

		tipCap, err := client.SuggestGasTipCap(c.Request().Context())
		if err != nil {
			return fmt.Errorf("failed to suggest tipcap: %w", err)
		}

		maxFee := new(big.Int).Mul(fees, new(big.Int).SetInt64(21000))
		tip := new(big.Int).Mul(tipCap, new(big.Int).SetInt64(21000))
		baseFee := new(big.Int).Sub(maxFee, tip)

		return c.JSON(http.StatusOK, output{
			EstimatedFees: maxFee.String(),
			BaseFee:       baseFee.String(),
			Tip:           tip.String(),
		})
	}
}
