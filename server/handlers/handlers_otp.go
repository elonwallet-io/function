package handlers

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"github.com/Leantar/elonwallet-function/models"
	"github.com/Leantar/elonwallet-function/server/common"
	"github.com/labstack/echo/v4"
	"math/big"
	"net/http"
	"strings"
	"time"
)

const invalidOTP = "The OTP provided is invalid or expired. Try creating a new OTP."

func (a *Api) CreateOTP() echo.HandlerFunc {
	return func(c echo.Context) error {
		user := c.Get("user").(models.User)

		otp, err := generateOTP()
		if err != nil {
			return fmt.Errorf("failed to generate otp: %w", err)
		}

		user.OTP = models.OTP{
			Secret:     otp,
			ValidUntil: time.Now().Add(time.Minute * 30).Unix(),
			TimesTried: 0,
			Active:     true,
		}

		err = a.repo.UpsertUser(user)
		if err != nil {
			return err
		}

		return c.NoContent(http.StatusCreated)
	}
}

func (a *Api) GetOTP() echo.HandlerFunc {
	return func(c echo.Context) error {
		user := c.Get("user").(models.User)

		if time.Now().After(time.Unix(user.OTP.ValidUntil, 0)) || user.OTP.TimesTried > 2 {
			user.OTP.Active = false
			err := a.repo.UpsertUser(user)
			if err != nil {
				return err
			}
		}

		return c.JSON(http.StatusOK, user.OTP)
	}
}

func (a *Api) LoginWithOTP() echo.HandlerFunc {
	type input struct {
		OTP string `json:"otp" validate:"is-otp"`
	}
	return func(c echo.Context) error {
		var in input
		if err := c.Bind(&in); err != nil {
			return err
		}
		if err := c.Validate(&in); err != nil {
			return err
		}

		user, err := a.repo.GetUser()
		if err != nil {
			return err
		}

		if !user.OTP.Active {
			return echo.NewHTTPError(http.StatusUnauthorized, invalidOTP)
		}
		if time.Now().After(time.Unix(user.OTP.ValidUntil, 0)) || user.OTP.TimesTried > 2 {
			user.OTP.Active = false
			err := a.repo.UpsertUser(user)
			if err != nil {
				return err
			}
			return echo.NewHTTPError(http.StatusUnauthorized, invalidOTP)
		}
		if user.OTP.Secret != in.OTP {
			user.OTP.TimesTried++
			err := a.repo.UpsertUser(user)
			if err != nil {
				return err
			}
			return echo.NewHTTPError(http.StatusUnauthorized, invalidOTP)
		}

		// Invalidate the otp after successful use
		user.OTP.Active = false
		err = a.repo.UpsertUser(user)
		if err != nil {
			return err
		}

		cookie, err := createOTPSessionCookie(user, a.signingKey.PrivateKey)
		if err != nil {
			return err
		}

		c.SetCookie(cookie)

		return c.NoContent(http.StatusOK)
	}
}

func generateOTP() (string, error) {
	var charset = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	var charsetLength = new(big.Int).SetInt64(int64(len(charset)))

	var sb strings.Builder
	for i := 0; i < 17; i++ {
		if i == 5 || i == 11 {
			sb.WriteString("-")
		} else {
			index, err := rand.Int(rand.Reader, charsetLength)
			if err != nil {
				return "", fmt.Errorf("failed to generate random char: %w", err)
			}
			sb.WriteRune(charset[index.Int64()])
		}
	}

	return sb.String(), nil
}

func createOTPSessionCookie(user models.User, sk ed25519.PrivateKey) (*http.Cookie, error) {
	jwt, err := common.CreateCredentialEnclaveJWT(user, sk)
	if err != nil {
		return nil, fmt.Errorf("failed to create jwt: %w", err)
	}

	return &http.Cookie{
		Name:     "session",
		Value:    jwt,
		Expires:  time.Now().Add(time.Minute * 15),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	}, nil
}
