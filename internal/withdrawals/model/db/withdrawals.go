package db

import (
	"gophermart/internal/utils"
	"gophermart/internal/withdrawals/model/api"
	"strconv"
	"time"
)

type Withdrawals struct {
	UserId      string    `db:"user_id"`
	Sum         int64     `db:"sum"`
	Number      int64     `db:"number"`
	ProcessedAt time.Time `db:"processed_at"`
}

func NewWithdrawals(userId string, sum float64, number int64) *Withdrawals {
	return &Withdrawals{UserId: userId, Sum: utils.GetPersistentAccrual(sum), Number: number}
}

func (w *Withdrawals) ToApi() api.Withdrawals {
	return api.Withdrawals{strconv.FormatInt(w.Number, 10), utils.GetApiAccrual(w.Sum), w.ProcessedAt}
}
