package model

import (
	"math"
	"time"
)

type OrderStatus int

/*
NEW — заказ загружен в систему, но не попал в обработку;
PROCESSING — вознаграждение за заказ рассчитывается;
INVALID — система расчёта вознаграждений отказала в расчёте;
PROCESSED — данные по заказу проверены и информация о расчёте успешно получена.
*/
const (
	New OrderStatus = iota
	Processing
	Invalid
	Processed
)

type Order struct {
	Number     int         `db:"number"`
	UserId     string      `db:"user_id"`
	Status     OrderStatus `db:"status"`
	UploadedAt time.Time   `db:"uploaded_at"`
	Accrual    int64       `db:"accrual"` //целая часть * 100 + дробная часть
}

func (order *Order) getAccrual() float64 {
	return float64(order.Accrual/100) + float64(order.Accrual%100)/100
}

func NewOrder(number int, userId string, status OrderStatus, accrual float64) *Order {
	integer, fraction := math.Modf(accrual)
	return &Order{Number: number, UserId: userId, Status: status, Accrual: int64(integer*100 + fraction*100)}
}
