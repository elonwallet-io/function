package models

type EmergencyAccessInvitation struct {
	GrantorEmail        string `json:"grantor_email"`
	WaitingPeriodInDays uint64 `json:"waiting_period_in_days"`
}
