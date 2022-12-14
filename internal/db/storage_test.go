package db

import (
	"context"
	accountModel "gophermart/internal/account/model/db"
	"gophermart/internal/order/model"
	withdrawalsModel "gophermart/internal/withdrawals/model/db"
	"testing"
	"time"

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
	xdb.MustExec("drop table if exists withdrawals;")
	xdb.MustExec("drop table if exists orders;")
	xdb.MustExec("drop table if exists accounts;")
	xdb.MustExec(`drop table if exists users;`)
}

func beforeTest() {
	xdb.MustExec("delete from withdrawals;")
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
		UserID  string
		order   uint64
		prepare func()
		check   func(error)
	}{
		{
			// ?????????? ?????????? ???????????? ???????????? ?? ??????????????????;
			name:   "save order success",
			UserID: "cfbe7630-32b3-11ed-a261-0242ac120002",
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
			// ?????????? ???????????? ?????? ?????? ???????????????? ???????? ??????????????????????????;
			name:   "save order failed: duplicate order",
			UserID: "cfbe7630-32b3-11ed-a261-0242ac120002",
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
			// ?????????? ???????????? ?????? ?????? ???????????????? ???????????? ??????????????????????????
			name:   "save order failed: another user's order",
			UserID: "cfbe7630-32b3-11ed-a261-0242ac120002",
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
					{Number: 1, UserID: "cfbe7630-32b3-11ed-a261-0242ac120002", Status: model.New, UploadedAt: uploadedAt, Accrual: 0},
					{Number: 2, UserID: "cfbe7630-32b3-11ed-a261-0242ac120002", Status: model.Processed, UploadedAt: uploadedAt, Accrual: 10},
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
				expected := &accountModel.Account{UserID: "cfbe7630-32b3-11ed-a261-0242ac120002", Current: 10, Withdrawn: 10}
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
				var UserID string
				assert.NoError(t, xdb.Get(&UserID, "select user_id from accounts where user_id = 'cfbe7630-32b3-11ed-a261-0242ac120002' and current = 500 and withdrawn = 1500"))
				assert.NoError(t, xdb.Get(&UserID, "select user_id from withdrawals where user_id = 'cfbe7630-32b3-11ed-a261-0242ac120002' and number = 1 and sum = 500"))
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

func Test_storageImpl_GetWithdrawals(t *testing.T) {
	db := initNewDB(t)
	tests := []struct {
		name    string
		prepare func()
		check   func([]withdrawalsModel.Withdrawals, error)
	}{
		{
			name: "GetWithdrawals successful",
			prepare: func() {
				xdb.MustExec(`insert into users(id, login, password) values('cfbe7630-32b3-11ed-a261-0242ac120002', 'login','password');`)
				xdb.MustExec(`insert into withdrawals(
					user_id,
					number,
					sum,
					processed_at) values
					(
						'cfbe7630-32b3-11ed-a261-0242ac120002',
						1,
						10,
						'2020-12-10T15:15:45+03:00'
					),
					(
						'cfbe7630-32b3-11ed-a261-0242ac120002',
						2,
						10,
						'2020-12-10T15:15:45+03:00'
					)
				`)
			},
			check: func(arr []withdrawalsModel.Withdrawals, err error) {
				tm, _ := time.Parse(time.RFC3339, "2020-12-10T15:15:45+03:00")
				processedAt := tm.In(arr[0].ProcessedAt.Location())
				expected := []withdrawalsModel.Withdrawals{
					{UserID: "cfbe7630-32b3-11ed-a261-0242ac120002", Sum: 10, Number: 1, ProcessedAt: processedAt},
					{UserID: "cfbe7630-32b3-11ed-a261-0242ac120002", Sum: 10, Number: 2, ProcessedAt: processedAt},
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
			tt.check(db.GetWithdrawals("cfbe7630-32b3-11ed-a261-0242ac120002"))
		})
	}
}

func Test_storageImpl_CalcAmounts(t *testing.T) {
	db := initNewDB(t)
	tests := []struct {
		name    string
		prepare func()
		updF    func(nums []int64) map[int64]CalcAmountsUpdateResult
		check   func(int, error)
		offset  int
		limit   int
	}{
		{
			name: "successful CalcAmounts",
			prepare: func() {
				xdb.MustExec(`insert into users(id, login, password) values('cfbe7630-32b3-11ed-a261-0242ac120002', 'login','password');`)
				xdb.MustExec(`insert into accounts(user_id, current, withdrawn) values('cfbe7630-32b3-11ed-a261-0242ac120002', 0, 0)`)
				xdb.MustExec(`
				insert into orders(
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
						1,
						'2020-12-10T15:15:45+03:00',
						0
					),
					(
						3,
						'cfbe7630-32b3-11ed-a261-0242ac120002',
						2,
						'2020-12-10T15:15:45+03:00',
						0
					),
					(
						4,
						'cfbe7630-32b3-11ed-a261-0242ac120002',
						3,
						'2020-12-10T15:15:45+03:00',
						0
					)
					`)
			},
			updF: func(nums []int64) map[int64]CalcAmountsUpdateResult {
				m := make(map[int64]CalcAmountsUpdateResult)
				m[1] = CalcAmountsUpdateResult{Accrual: 10, Status: 3}
				m[2] = CalcAmountsUpdateResult{Accrual: 10, Status: 3}

				return m
			},
			check: func(updated int, err error) {
				assert.Equal(t, 2, updated)
				assert.NoError(t, err)
				var n int
				assert.NoError(t, xdb.Get(&n, "select count(1) from orders where number in (1,2) and status = 3 and accrual = 10"))
				assert.Equal(t, 2, n)
				assert.NoError(t, xdb.Get(&n, "select count(1) from accounts where user_id = 'cfbe7630-32b3-11ed-a261-0242ac120002' and current = 20"))
				assert.Equal(t, 1, n)
			},
			offset: 0,
			limit:  10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beforeTest()
			tt.prepare()
			tt.check(db.CalcAmounts(tt.offset, tt.limit, tt.updF))
		})
	}
}
