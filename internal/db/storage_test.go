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
	xdb.MustExec(`drop table if exists users;`)
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
			cleanUsersTable()
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
			cleanUsersTable()
			tt.prepare()
			tt.check(db.GetByLoginPassword(tt.args.login, tt.args.password))
		})
	}
}
