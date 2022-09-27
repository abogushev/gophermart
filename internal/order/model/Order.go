package model

import (
	"gophermart/internal/order/model/api"
	"gophermart/internal/utils"
	"strconv"
	"time"
)

/*
NEW — заказ загружен в систему, но не попал в обработку;
PROCESSING — вознаграждение за заказ рассчитывается;
INVALID — система расчёта вознаграждений отказала в расчёте;
PROCESSED — данные по заказу проверены и информация о расчёте успешно получена.
*/
type OrderStatus int

const (
	New OrderStatus = iota
	Processing
	Invalid
	Processed
)

type Order struct {
	Number     uint64      `db:"number"`
	UserID     string      `db:"user_id"`
	Status     OrderStatus `db:"status"`
	UploadedAt time.Time   `db:"uploaded_at"`
	Accrual    int64       `db:"accrual"` //целая часть * 100 + дробная часть
}

func NewOrder(number uint64, UserID string, status OrderStatus, accrual float64) *Order {
	return &Order{Number: number, UserID: UserID, Status: status, Accrual: utils.GetPersistentAccrual(accrual)}
}

func (o *Order) ToAPI() api.Order {
	s := ""
	accrual := utils.GetAPIAccrual(o.Accrual)

	switch o.Status {
	case New:
		s = api.New
	case Processing:
		s = api.Processing
	case Invalid:
		s = api.Invalid
	case Processed:
		s = api.Processed
	}

	var ac *float64
	if o.Status == Processed {
		ac = &accrual
	}

	return api.Order{Number: strconv.FormatUint(o.Number, 10), UserID: o.UserID, Status: s, UploadedAt: o.UploadedAt, Accrual: ac}
}
