package api

import "time"

type Order struct {
	Number     int       `json:"number"`
	UserId     string    `json:"user_id"`
	Status     string    `json:"status"`
	UploadedAt time.Time `json:"uploaded_at"`
	Accrual    float64   `json:"accrual,omitempty"`
}
