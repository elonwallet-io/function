package server

import (
	"crypto/ed25519"
	"github.com/Leantar/elonwallet-function/models"
	"github.com/labstack/echo/v4"
	"net/http"
)

func (a *Api) HandleGetJWTVerificationKey() echo.HandlerFunc {
	type output struct {
		VerificationKey []byte `json:"verification_key"`
	}
	return func(c echo.Context) error {
		user := c.Get("user").(*models.User)

		pk := user.JWTSigningKey.Public().(ed25519.PublicKey)

		return c.JSON(http.StatusOK, output{
			VerificationKey: pk,
		})
	}

}
