package handlers

import (
	"github.com/Leantar/elonwallet-function/models"
	"github.com/labstack/echo/v4"
	"net/http"
)

func (a *Api) HandleGetWallets() echo.HandlerFunc {
	type redactedWallet struct {
		Name    string `json:"name"`
		Address string `json:"address"`
		Public  bool   `json:"public"`
	}
	type output struct {
		Wallets []redactedWallet `json:"wallets"`
	}
	return func(c echo.Context) error {
		user := c.Get("user").(models.User)

		redactedWallets := make([]redactedWallet, len(user.Wallets))
		for i, wallet := range user.Wallets {
			redactedWallets[i] = redactedWallet{
				Name:    wallet.Name,
				Address: wallet.Address,
				Public:  wallet.Public,
			}
		}

		return c.JSON(http.StatusOK, output{
			Wallets: redactedWallets,
		})
	}
}

func (a *Api) HandleCreateWallet() echo.HandlerFunc {
	type input struct {
		Name   string `json:"name" validate:"required,alphanum"`
		Public bool   `json:"public"`
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

		if user.Wallets.Exists(in.Name) {
			return echo.NewHTTPError(http.StatusBadRequest, "A wallet with this name already exists")
		}

		wallet, err := a.createWallet(in.Name, in.Public, user)
		if err != nil {
			return err
		}
		user.Wallets = append(user.Wallets, wallet)

		err = a.repo.UpsertUser(user)
		if err != nil {
			return err
		}

		return c.NoContent(http.StatusCreated)
	}
}
