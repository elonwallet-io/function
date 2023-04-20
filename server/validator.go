package server

import (
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"net/http"
	"regexp"
)

const (
	isOTPRegexString = "^[A-Z0-9]{5}-[A-Z0-9]{5}-[A-Z0-9]{5}$"
)

var (
	isOTPRegex = regexp.MustCompile(isOTPRegexString)
)

type CustomValidator struct {
	validator *validator.Validate
}

func (c *CustomValidator) Validate(i interface{}) error {
	if err := c.validator.Struct(i); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return nil
}

func newValidator() CustomValidator {
	v := CustomValidator{
		validator: validator.New(),
	}

	_ = v.validator.RegisterValidation("is-otp", ValidateOTP, false)

	return v
}

func ValidateOTP(fl validator.FieldLevel) bool {
	return isOTPRegex.MatchString(fl.Field().String())
}
