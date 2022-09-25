package db

import (
	"context"
	"database/sql"
	"errors"
	"gophermart/internal/order/model"
	"gophermart/internal/utils"

	accountModel "gophermart/internal/account/model/db"
	withdrawalsModel "gophermart/internal/withdrawals/model/db"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

type CalcAmountsUpdateResult struct {
	Accrual int64
	Status  model.OrderStatus
}
type Storage interface {
	Register(login, password string) (string, error)
	GetByLoginPassword(login, password string) (string, error)
	SaveOrder(UserID string, number uint64) error
	GetOrders(UserID string) ([]model.Order, error)

	GetAccount(UserID string) (*accountModel.Account, error)
	WithdrawFromAccount(UserID string, sum float64, number uint64) error
	GetWithdrawals(UserID string) ([]withdrawalsModel.Withdrawals, error)
	CalcAmounts(offset, limit int, updF func(nums []int64) map[int64]CalcAmountsUpdateResult) (int, error)
}

var ErrDuplicateLogin = errors.New("login already exist")
var ErrUserNotFound = errors.New("user not found")

var ErrDuplicateOrder = errors.New("the order number has already been uploaded by this user")
var ErrOrderOfAnotherUser = errors.New("the order number has already been uploaded by another user")

var ErrBalanceLimitExhausted = errors.New("there are not enough funds in the account")

type storageImpl struct {
	url    string
	ctx    context.Context
	xdb    *sqlx.DB
	logger *zap.SugaredLogger
}

const (
	createTablesIfNeedSQL = `
	create table if not exists users(
		id uuid primary key, 
		login varchar(256) unique, 
		password varchar(256) not null
	);

	create table if not exists orders (
		number bigint primary key,
		user_id UUID,
		status int not null default 0,
		uploaded_at timestamp with time zone not null default now(),
		accrual integer not null default 0,
		CONSTRAINT fk_user
		FOREIGN KEY(user_id) 
		REFERENCES users(id)
	);

	create table if not exists accounts(
		user_id UUID,
		current integer not null default 0,
		withdrawn integer not null default 0,
		CONSTRAINT fk_user
		FOREIGN KEY(user_id) 
		REFERENCES users(id)
	);

	create table if not exists withdrawals(
		user_id UUID,
		number bigint not null unique,
		sum integer not null,
		processed_at timestamp with time zone not null default now(),
		CONSTRAINT fk_user
		FOREIGN KEY(user_id) 
		REFERENCES users(id)
	);
	`

	getUserIDByLoginPasswordSQL = `select id from users where login = $1 and password = $2;`
	getCountByLoginPasswordSQL  = `select count(*) from users where login = $1 and password = $2;`
	insertUserSQL               = `insert into users(id, login, password) values($1,$2,$3);`

	getOrderUserIDSQL          = `select user_id from orders where number = $1;`
	saveOrderSQL               = `insert into orders(user_id, number) values($1,$2);`
	selectAllOrdersOfUserIDSQL = `
	select
		number,
		status,
		user_id,
		accrual,
		uploaded_at 
	from orders where user_id = $1 order by uploaded_at asc;`

	getUserAccount                  = `select user_id, current, withdrawn from accounts where user_id = $1`
	getUserAccountForUpdate         = `select user_id, current, withdrawn from accounts where user_id = $1 for update`
	updateAccount                   = `update accounts set current = $2, withdrawn = $3 where user_id = $1`
	insertWithdrawals               = `insert into withdrawals(user_id,number,sum) values($1,$2,$3);`
	selectAllwithdrawalsOfUserIDSQL = `select user_id,number,sum,processed_at from withdrawals where user_id = $1 order by processed_at asc`

	selectOrdersForCalc         = `select number from orders where status = 0 or status = 1 offset $1 limit $2 for update`
	updateOrdersForCalc         = `update orders set status = $2, accrual = $3 where number = $1`
	selectAccountAccuralForCalc = `select user_id, sum(accrual) from orders where number in ($1) group by user_id;`
	addAccountAccuralForCalc    = `update accounts set current = current + $2 where user_id = $1`
)

func NewStorage(url string, ctx context.Context, logger *zap.SugaredLogger) (Storage, error) {
	logger.Infow("start init dbstorage ...")
	xdb, err := sqlx.Connect("postgres", url)
	if err != nil {
		logger.Errorf("error on connect to db: %v", err)
		return nil, err
	}

	storage := &storageImpl{url, ctx, xdb, logger}
	if err := storage.initDB(); err != nil {
		logger.Errorf("error on connect to init db: %v", err)
		return nil, err
	}
	logger.Info("dbstorage initialized successfully")
	return storage, nil
}

func (db *storageImpl) initDB() error {
	_, err := db.xdb.ExecContext(db.ctx, createTablesIfNeedSQL)
	return err
}

func (db *storageImpl) Register(login, password string) (string, error) {
	tx, err := db.xdb.Beginx()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	var count int
	if err := db.xdb.GetContext(db.ctx, &count, getCountByLoginPasswordSQL, login, password); err != nil {
		return "", err
	}
	if count > 0 {
		return "", ErrDuplicateLogin
	}
	id := uuid.New().String()
	if _, err := db.xdb.ExecContext(db.ctx, insertUserSQL, id, login, password); err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	return id, nil
}

func (db *storageImpl) GetByLoginPassword(login, password string) (string, error) {
	var id string
	err := db.xdb.GetContext(db.ctx, &id, getUserIDByLoginPasswordSQL, login, password)
	if err == sql.ErrNoRows {
		return "", ErrUserNotFound
	} else if err != nil {
		return "", err
	}

	return id, nil
}

func (db *storageImpl) SaveOrder(UserID string, number uint64) error {
	tx, err := db.xdb.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var orderUserID string
	err = db.xdb.GetContext(db.ctx, &orderUserID, getOrderUserIDSQL, number)

	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if err == nil {
		if orderUserID == UserID {
			return ErrDuplicateOrder
		}
		return ErrOrderOfAnotherUser
	}

	if _, err = db.xdb.ExecContext(db.ctx, saveOrderSQL, UserID, number); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (db *storageImpl) GetOrders(UserID string) ([]model.Order, error) {
	orders := []model.Order{}
	if err := db.xdb.SelectContext(db.ctx, &orders, selectAllOrdersOfUserIDSQL, UserID); err != nil {
		return nil, err
	}
	return orders, nil
}

//Account

func (db *storageImpl) GetAccount(UserID string) (*accountModel.Account, error) {
	var account accountModel.Account
	err := db.xdb.GetContext(db.ctx, &account, getUserAccount, UserID)
	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	} else if err != nil {
		return nil, err
	}

	return &account, nil
}

func (db *storageImpl) WithdrawFromAccount(UserID string, sum float64, number uint64) error {
	tx, err := db.xdb.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var acc accountModel.Account
	err = db.xdb.GetContext(db.ctx, &acc, getUserAccountForUpdate, UserID)
	if err == sql.ErrNoRows {
		return ErrUserNotFound
	} else if err != nil {
		return err
	}

	withdraw := utils.GetPersistentAccrual(sum)
	if acc.Current < withdraw {
		return ErrBalanceLimitExhausted
	}
	newCurrent := acc.Current - withdraw
	newWithdrawn := acc.Withdrawn + withdraw

	if _, err := db.xdb.ExecContext(db.ctx, updateAccount, acc.UserID, newCurrent, newWithdrawn); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (db *storageImpl) GetWithdrawals(UserID string) ([]withdrawalsModel.Withdrawals, error) {
	withdrawals := []withdrawalsModel.Withdrawals{}
	if err := db.xdb.SelectContext(db.ctx, &withdrawals, selectAllwithdrawalsOfUserIDSQL, UserID); err != nil {
		return nil, err
	}
	return withdrawals, nil
}

func (db *storageImpl) CalcAmounts(offset, limit int, updF func(nums []int64) map[int64]CalcAmountsUpdateResult) (int, error) {
	tx, err := db.xdb.Beginx()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	nums := []int64{}
	if err := db.xdb.SelectContext(db.ctx, &nums, selectOrdersForCalc, offset, limit); err != nil {
		return 0, err
	}

	if len(nums) != 0 {
		updates := updF(nums)
		for num, accAndStatus := range updates {
			if _, err := db.xdb.ExecContext(db.ctx, updateOrdersForCalc, num, accAndStatus.Status, accAndStatus.Accrual); err != nil {
				if err := tx.Rollback(); err != nil {
					return 0, err
				}
				return 0, err
			}
		}
		userIdUpd := make([]struct {
			userId string `db:"user_id"`
			sum    int64  `db:"sum"`
		}, 0)
		if err := db.xdb.SelectContext(db.ctx, userIdUpd, selectAccountAccuralForCalc, nums); err != nil {
			return 0, err
		}
		for i := 0; i < len(userIdUpd); i++ {
			if _, err := db.xdb.ExecContext(db.ctx, addAccountAccuralForCalc, userIdUpd[i].userId, userIdUpd[i].sum); err != nil {
				return 0, err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return len(nums), nil
}
