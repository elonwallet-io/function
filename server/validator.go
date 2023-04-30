package server

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"net/http"
	"regexp"
)

const (
	isOTPRegexString             = "^[A-Z0-9]{5}-[A-Z0-9]{5}-[A-Z0-9]{5}$"
	isEIP55EthAddressRegexString = "^0x[0-9a-fA-F]{40}$"
	isOtherEthAddressRegexString = "(^0x[0-9a-f]{40}$)|(^0x[0-9A-F]{40}$)"
)

var (
	isOTPRegex             = regexp.MustCompile(isOTPRegexString)
	isEIP55EthAddressRegex = regexp.MustCompile(isEIP55EthAddressRegexString)
	isOtherEthAddressRegex = regexp.MustCompile(isOtherEthAddressRegexString)
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

	_ = v.validator.RegisterValidation("otp", ValidateOTP, false)
	_ = v.validator.RegisterValidation("ethereum_address", ValidateEthereumAddress, false)

	return v
}

func ValidateOTP(fl validator.FieldLevel) bool {
	return isOTPRegex.MatchString(fl.Field().String())
}

func ValidateEthereumAddress(fl validator.FieldLevel) (valid bool) {
	value := fl.Field().String()

	if isOtherEthAddressRegex.MatchString(value) {
		valid = true
	} else if isEIP55EthAddressRegex.MatchString(value) {
		// Validate if the checksum hash matches the provided value
		// See https://eips.ethereum.org/EIPS/eip-55 for more info
		addr := common.HexToAddress(value)
		valid = addr.Hex() == value
	}

	return
}
