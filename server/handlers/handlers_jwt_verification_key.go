package handlers

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

func (a *Api) HandleGetJWTVerificationKey() echo.HandlerFunc {
	type output struct {
		VerificationKey []byte `json:"verification_key"`
	}
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, output{
			VerificationKey: a.signingKey.PublicKey,
		})
	}

}
