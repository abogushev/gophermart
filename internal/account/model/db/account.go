package db

import (
	"gophermart/internal/account/model/api"
	"gophermart/internal/utils"
)

type Account struct {
	UserID    string `db:"user_id"`
	Current   int64  `db:"current"`
	Withdrawn int64  `db:"withdrawn"`
}

func (a *Account) ToAPI() api.Account {
	return api.Account{Current: utils.GetAPIAccrual(a.Current), Withdrawn: utils.GetAPIAccrual(a.Withdrawn)}
}
