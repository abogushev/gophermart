package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

type Storage interface {
	Register(login, password string) (string, error)
	GetByLoginPassword(login, password string) (string, error)
	// SaveOrder(login string, order int) error
}

var ErrDuplicateLogin = errors.New("login already exist")
var ErrUserNotFound = errors.New("user not found")

var ErrDuplicateOrder = errors.New("the order number has already been uploaded by this user")
var ErrOrderOfAnotherUser = errors.New("the order number has already been uploaded by another user")

type storageImpl struct {
	url    string
	ctx    context.Context
	xdb    *sqlx.DB
	logger *zap.SugaredLogger
}

const (
	createUserTableIfNeedSQL = `create table if not exists users(
		id uuid primary key, 
		login varchar(256) unique, 
		password varchar(256) not null
	);`

	getUserIdByLoginPasswordSQL = `select id from users where login = $1 and password = $2;`
	getCountByLoginPasswordSQL  = `select count(*) from users where login = $1 and password = $2;`
	insertUserSQL               = `insert into users(id, login, password) values($1,$2,$3) returning id;`

	createOrderTableIfNeedSQL = `create table if not exists orders (
		number integer primary key,
		userId UUID, 
		CONSTRAINT fk_user
		FOREIGN KEY(userId) 
		REFERENCES users(id)
	);`
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
	_, err := db.xdb.ExecContext(db.ctx, createUserTableIfNeedSQL)
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
	row := db.xdb.QueryRowContext(db.ctx, insertUserSQL, uuid.New().String(), login, password)

	var id string
	if err := row.Scan(&id); err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	return id, nil
}

func (db *storageImpl) GetByLoginPassword(login, password string) (string, error) {
	var id string
	err := db.xdb.GetContext(db.ctx, &id, getUserIdByLoginPasswordSQL, login, password)
	if err == sql.ErrNoRows {
		return "", ErrUserNotFound
	} else if err != nil {
		return "", err
	}

	return id, nil
}
