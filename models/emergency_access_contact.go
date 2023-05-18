package models

import "crypto/ed25519"

type EmergencyAccessContact struct {
	Email                string            `json:"email"`
	EnclaveURL           string            `json:"enclave_url"`
	JWTSigningKey        ed25519.PublicKey `json:"jwt_signing_key"`
	HasAccepted          bool              `json:"has_accepted"`
	HasRequestedTakeover bool              `json:"has_requested_takeover"`
	WaitingPeriodInDays  uint64            `json:"waiting_period_in_days"`
	TakeoverAllowedAfter int64             `json:"takeover_allowed_after"`
}
