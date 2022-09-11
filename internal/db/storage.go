package db

import (
	"context"
	"errors"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

type Storage interface {
	Register(login, password string) error
	IsExist(login, password string) (bool, error)
}

var ErrDuplicateLogin = errors.New("login already exist")

type storageImpl struct {
	url    string
	ctx    context.Context
	xdb    *sqlx.DB
	logger *zap.SugaredLogger
}

const (
	createTableIfNeedSQL         = `create table if not exists users(login varchar(256) primary key, password varchar(256) not null) ;`
	countUsersByLoginPasswordSQL = `select count(*) from users where login = $1 and password = $2;`
	insertUserSQL                = `insert into users(login, password) values($1,$2);`
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
	_, err := db.xdb.ExecContext(db.ctx, createTableIfNeedSQL)
	return err
}

func (db *storageImpl) Register(login, password string) error {
	tx, err := db.xdb.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var count int
	if err := db.xdb.GetContext(db.ctx, &count, countUsersByLoginPasswordSQL, login, password); err != nil {
		return err
	}
	if count > 0 {
		return ErrDuplicateLogin
	}
	if _, err := db.xdb.ExecContext(db.ctx, insertUserSQL, login, password); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (db *storageImpl) IsExist(login, password string) (bool, error) {
	var count int
	if err := db.xdb.GetContext(db.ctx, &count, countUsersByLoginPasswordSQL, login, password); err != nil {
		return false, err
	}
	return count > 0, nil
}
