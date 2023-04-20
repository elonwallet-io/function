package models

type OTP struct {
	Secret     string `json:"secret"`
	ValidUntil int64  `json:"valid_until"`
	TimesTried int64  `json:"times_tried"`
	Active     bool   `json:"active"`
}
