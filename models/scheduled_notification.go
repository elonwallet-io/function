package models

type ScheduledNotification struct {
	SendAfter int64  `json:"send_after"`
	Title     string `json:"title"`
	Body      string `json:"body"`
}
