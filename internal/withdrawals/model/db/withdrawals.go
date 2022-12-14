package db

import (
	"gophermart/internal/utils"
	"gophermart/internal/withdrawals/model/api"
	"strconv"
	"time"
)

type Withdrawals struct {
	UserID      string    `db:"user_id"`
	Sum         int64     `db:"sum"`
	Number      uint64    `db:"number"`
	ProcessedAt time.Time `db:"processed_at"`
}

func NewWithdrawals(UserID string, sum float64, number uint64) *Withdrawals {
	return &Withdrawals{UserID: UserID, Sum: utils.GetPersistentAccrual(sum), Number: number}
}

func (w *Withdrawals) ToAPI() api.Withdrawals {
	return api.Withdrawals{Order: strconv.FormatUint(w.Number, 10), Sum: utils.GetAPIAccrual(w.Sum), ProcessedAt: w.ProcessedAt}
}
