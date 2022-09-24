package db

import (
	"gophermart/internal/account/model/api"
)

type Account struct {
	UserID    string `db:"user_id"`
	Current   int64  `db:"current"`
	Withdrawn int64  `db:"withdrawn"`
}

func (a *Account) ToAPI() api.Account {
	return api.Account{Current: a.Current, Withdrawn: a.Withdrawn}
}
