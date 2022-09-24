package api

import "time"

type Order struct {
	Number     string    `json:"number"`
	UserID     string    `json:"user_id"`
	Status     string    `json:"status"`
	UploadedAt time.Time `json:"uploaded_at"`
	Accrual    *float64  `json:"accrual,omitempty"`
}
