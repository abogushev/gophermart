package db

import (
	"context"
	"gophermart/internal/order/model"
	"testing"
	"time"

	accountModel "gophermart/internal/account/model/db"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func getLogger() *zap.SugaredLogger {
	l, _ := zap.NewProduction()
	return l.Sugar()
}

var connURL = "host=localhost port=5432 user=postgres dbname=postgres sslmode=disable"
var xdb = sqlx.MustConnect("postgres", connURL)

func dropTables() {
	xdb.MustExec("drop table if exists orders;")
	xdb.MustExec("drop table if exists accounts;")
	xdb.MustExec(`drop table if exists users;`)
}

func beforeTest() {
	xdb.MustExec(`delete from orders;`)
	xdb.MustExec(`delete from accounts;`)
	xdb.MustExec(`delete from users;`)
}

func initNewDB(t *testing.T) Storage {
	dropTables()
	if db, err := NewStorage(connURL, context.TODO(), getLogger()); err != nil {
		t.Fatal(err)
		return nil
	} else {
		return db
	}
}

func Test_storageImpl_Register(t *testing.T) {
	db := initNewDB(t)
	type args struct {
		login    string
		password string
	}
	tests := []struct {
		name    string
		args    args
		prepare func()
		check   func(string, error)
	}{
		{
			"register success",
			args{login: "login", password: "password"},
			func() {},
			func(id string, err error) {
				assert.NotEmpty(t, id, "id is empty")
				assert.NoError(t, err, "error not eq nil")
			},
		},
		{
			"duplicate login",
			args{login: "login", password: "password"},
			func() {
				xdb.MustExec(`insert into users(id, login, password) values('cfbe7630-32b3-11ed-a261-0242ac120002','login','password');`)
			},
			func(id string, err error) {
				assert.Empty(t, id, "id must be empty")
				assert.ErrorIs(t, err, ErrDuplicateLogin)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beforeTest()
			tt.prepare()
			tt.check(db.Register(tt.args.login, tt.args.password))
		})
	}
}

func Test_storageImpl_GetByLoginPassword(t *testing.T) {
	db := initNewDB(t)
	type args struct {
		login    string
		password string
	}
	tests := []struct {
		name    string
		args    args
		prepare func()
		check   func(string, error)
	}{
		{
			"GetByLoginPassword success",
			args{login: "login", password: "password"},
			func() {
				xdb.MustExec(`insert into users(id, login, password) values('cfbe7630-32b3-11ed-a261-0242ac120002', 'login','password');`)
			},
			func(id string, err error) {
				assert.NotEmpty(t, id, "id must be not empty")
				assert.NoError(t, err, "error not eq nil")
			},
		},
		{
			"GetByLoginPassword failed",
			args{login: "login", password: "password"},
			func() {},
			func(id string, err error) {
				assert.Empty(t, id, "id must be empty")
				assert.ErrorIs(t, err, ErrUserNotFound)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beforeTest()
			tt.prepare()
			tt.check(db.GetByLoginPassword(tt.args.login, tt.args.password))
		})
	}
}

func Test_storageImpl_SaveOrder(t *testing.T) {
	db := initNewDB(t)

	tests := []struct {
		name    string
		userId  string
		order   int
		prepare func()
		check   func(error)
	}{
		{
			// новый номер заказа принят в обработку;
			name:   "save order success",
			userId: "cfbe7630-32b3-11ed-a261-0242ac120002",
			order:  1,
			prepare: func() {
				xdb.MustExec(`insert into users(id, login, password) values('cfbe7630-32b3-11ed-a261-0242ac120002', 'login','password');`)
			},
			check: func(err error) {
				assert.NoError(t, err, "err not eq nil")
				var number int
				assert.NoError(t, xdb.Get(&number, "select number from orders where number = 1 and user_id = 'cfbe7630-32b3-11ed-a261-0242ac120002'"))
			},
		},
		{
			// номер заказа уже был загружен этим пользователем;
			name:   "save order failed: duplicate order",
			userId: "cfbe7630-32b3-11ed-a261-0242ac120002",
			order:  1,
			prepare: func() {
				xdb.MustExec(`insert into users(id, login, password) values('cfbe7630-32b3-11ed-a261-0242ac120002', 'login','password');`)
				xdb.MustExec("insert into orders(number, user_id) values(1, 'cfbe7630-32b3-11ed-a261-0242ac120002');")
			},
			check: func(err error) {
				assert.ErrorIs(t, err, ErrDuplicateOrder, "must be duplicate order err")
			},
		},
		{
			// номер заказа уже был загружен другим пользователем
			name:   "save order failed: another user's order",
			userId: "cfbe7630-32b3-11ed-a261-0242ac120002",
			order:  1,
			prepare: func() {
				xdb.MustExec(`insert into users(id, login, password) values('cfbe7630-32b3-11ed-a261-0242ac120002', 'login','password');`)
				xdb.MustExec(`insert into users(id, login, password) values('cfbe7630-32b3-11ed-a261-0242ac120003', 'login2','password2');`)
				xdb.MustExec("insert into orders(number, user_id) values(1, 'cfbe7630-32b3-11ed-a261-0242ac120003');")
			},
			check: func(err error) {
				assert.ErrorIs(t, err, ErrOrderOfAnotherUser, "must be order of another user err")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beforeTest()
			tt.prepare()
			tt.check(db.SaveOrder("cfbe7630-32b3-11ed-a261-0242ac120002", tt.order))
		})
	}
}

func Test_storageImpl_GetOrders(t *testing.T) {
	db := initNewDB(t)
	tests := []struct {
		name    string
		prepare func()
		check   func([]model.Order, error)
	}{
		{
			name: "get full orders of user",
			prepare: func() {
				xdb.MustExec(`insert into users(id, login, password) values('cfbe7630-32b3-11ed-a261-0242ac120002', 'login','password');`)
				xdb.MustExec(`insert into orders(
					number,
					user_id,
					status,
					uploaded_at,
					accrual) values
					(
						1,
						'cfbe7630-32b3-11ed-a261-0242ac120002',
						0,
						'2020-12-10T15:15:45+03:00',
						0
					),
					(
						2,
						'cfbe7630-32b3-11ed-a261-0242ac120002',
						3,
						'2020-12-10T15:15:45+03:00',
						10
					)
				`)
			},
			check: func(arr []model.Order, err error) {
				tm, _ := time.Parse(time.RFC3339, "2020-12-10T15:15:45+03:00")
				uploadedAt := tm.In(arr[0].UploadedAt.Location())
				expected := []model.Order{
					{Number: 1, UserId: "cfbe7630-32b3-11ed-a261-0242ac120002", Status: model.New, UploadedAt: uploadedAt, Accrual: 0},
					{Number: 2, UserId: "cfbe7630-32b3-11ed-a261-0242ac120002", Status: model.Processed, UploadedAt: uploadedAt, Accrual: 10},
				}
				assert.NoError(t, err)

				assert.Equal(t, expected, arr)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beforeTest()
			tt.prepare()
			tt.check(db.GetOrders("cfbe7630-32b3-11ed-a261-0242ac120002"))
		})
	}
}

func Test_storageImpl_GetAccount(t *testing.T) {
	db := initNewDB(t)
	tests := []struct {
		name    string
		prepare func()
		check   func(*accountModel.Account, error)
	}{
		{
			name: "get account",
			prepare: func() {
				xdb.MustExec(`insert into users(id, login, password) values('cfbe7630-32b3-11ed-a261-0242ac120002', 'login','password');`)
				xdb.MustExec(`insert into accounts(user_id, current, withdrawn) values('cfbe7630-32b3-11ed-a261-0242ac120002', 10, 10)`)
			},
			check: func(acc *accountModel.Account, err error) {
				assert.NoError(t, err)
				expected := &accountModel.Account{"cfbe7630-32b3-11ed-a261-0242ac120002", 10, 10}
				assert.Equal(t, expected, acc)
			},
		},
		{
			name: "not exist account",
			prepare: func() {
				xdb.MustExec(`insert into users(id, login, password) values('cfbe7630-32b3-11ed-a261-0242ac120002', 'login','password');`)
			},
			check: func(acc *accountModel.Account, err error) {
				assert.ErrorIs(t, err, ErrUserNotFound)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beforeTest()
			tt.prepare()
			tt.check(db.GetAccount("cfbe7630-32b3-11ed-a261-0242ac120002"))
		})
	}
}

func Test_storageImpl_WithdrawFromAccount(t *testing.T) {
	db := initNewDB(t)
	tests := []struct {
		name    string
		sum     float64
		prepare func()
		check   func(error)
	}{
		{
			name: "successfull withdraw",
			sum:  5.0,
			prepare: func() {
				xdb.MustExec(`insert into users(id, login, password) values('cfbe7630-32b3-11ed-a261-0242ac120002', 'login','password');`)
				xdb.MustExec(`insert into accounts(user_id, current, withdrawn) values('cfbe7630-32b3-11ed-a261-0242ac120002', 1000, 1000)`)
			},
			check: func(err error) {
				assert.NoError(t, err)
				var userId string
				assert.NoError(t, xdb.Get(&userId, "select user_id from accounts where user_id = 'cfbe7630-32b3-11ed-a261-0242ac120002' and current = 500 and withdrawn = 1500"))
			},
		},
		{
			name: "user not found",
			sum:  5.0,
			prepare: func() {
				xdb.MustExec(`insert into users(id, login, password) values('cfbe7630-32b3-11ed-a261-0242ac120003', 'login','password');`)
				xdb.MustExec(`insert into accounts(user_id, current, withdrawn) values('cfbe7630-32b3-11ed-a261-0242ac120003', 1000, 1000)`)
			},
			check: func(err error) {
				assert.ErrorIs(t, err, ErrUserNotFound)
			},
		},
		{
			name: "fail withdraw: out of limit",
			sum:  5.0,
			prepare: func() {
				xdb.MustExec(`insert into users(id, login, password) values('cfbe7630-32b3-11ed-a261-0242ac120002', 'login','password');`)
				xdb.MustExec(`insert into accounts(user_id, current, withdrawn) values('cfbe7630-32b3-11ed-a261-0242ac120002', 10, 10)`)
			},
			check: func(err error) {
				assert.ErrorIs(t, err, ErrBalanceLimitExhausted)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beforeTest()
			tt.prepare()
			tt.check(db.WithdrawFromAccount("cfbe7630-32b3-11ed-a261-0242ac120002", tt.sum, 1))
		})
	}
}
