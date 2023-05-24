package models

type EmergencyAccessContact struct {
	Email                string `json:"email"`
	EnclaveURL           string `json:"enclave_url"`
	HasAccepted          bool   `json:"has_accepted"`
	HasRequestedTakeover bool   `json:"has_requested_takeover"`
	WaitingPeriodInDays  uint64 `json:"waiting_period_in_days"`
	TakeoverAllowedAfter int64  `json:"takeover_allowed_after"`
	NotificationSeriesID string `json:"notification_series_id"`
}
