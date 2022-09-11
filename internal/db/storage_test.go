package db

import (
	"context"
	"testing"

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
	xdb.MustExec(`drop table users;`)
}

func cleanUsersTable() {
	xdb.MustExec(`truncate table users;`)
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
		check   func(error)
	}{
		{
			"register success",
			args{login: "login", password: "password"},
			func() {},
			func(err error) { assert.NoError(t, err, "error not eq nil") },
		},
		{
			"duplicate login",
			args{login: "login", password: "password"},
			func() {

				xdb.MustExec(`insert into users(login, password) values('login','password');`)
			},
			func(err error) {
				assert.ErrorIs(t, err, ErrDuplicateLogin)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanUsersTable()
			tt.prepare()
			tt.check(db.Register(tt.args.login, tt.args.password))
		})
	}
}

func Test_storageImpl_IsExist(t *testing.T) {
	db := initNewDB(t)
	type args struct {
		login    string
		password string
	}
	tests := []struct {
		name    string
		args    args
		prepare func()
		check   func(bool, error)
	}{
		{
			"IsExist = true",
			args{login: "login", password: "password"},
			func() {
				xdb.MustExec(`insert into users(login, password) values('login','password');`)
			},
			func(isExist bool, err error) {
				assert.True(t, isExist, "failed to find user")
				assert.NoError(t, err, "error not eq nil")
			},
		},
		{
			"IsExist = false",
			args{login: "login", password: "password"},
			func() {},
			func(isExist bool, err error) {
				assert.False(t, isExist, "find unexpected user")
				assert.NoError(t, err, "error not eq nil")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanUsersTable()
			tt.prepare()
			tt.check(db.IsExist(tt.args.login, tt.args.password))
		})
	}
}
