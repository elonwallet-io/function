package models

type EmergencyAccessGrant struct {
	Email                string `json:"email"`
	EnclaveURL           string `json:"enclave_url"`
	HasAccepted          bool   `json:"has_accepted"`
	HasRequestedTakeover bool   `json:"has_requested_takeover"`
	TakeoverAllowedAfter int64  `json:"takeover_allowed_after"`
	NotificationSeriesID string `json:"notification_series_id"`
}
