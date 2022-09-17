package auth

import (
	"bytes"
	"errors"
	"fmt"
	"gophermart/internal/db"
	"gophermart/internal/order/model"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockDbStorage struct {
	mock.Mock
}

func (m *mockDbStorage) Register(login string, password string) (string, error) {
	args := m.Called(login, password)
	return args.String(0), args.Error(1)
}

func (m *mockDbStorage) GetByLoginPassword(login string, password string) (string, error) {
	args := m.Called(login, password)
	return args.String(0), args.Error(1)
}

func (m *mockDbStorage) SaveOrder(userId string, number int) error {
	return nil
}

func (m *mockDbStorage) GetOrders(userId string) ([]model.Order, error) {
	return nil, nil
}

var secret = "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJMb2dpbiI6ImxvZ2luIn0.cJ-fGT2jF6lVw1dF6MfN7k44KuNGdRowac6RXzCFO997Sjo0Uk_wNVtj2i8jtUt9_0RQI1CnsHu5dOcINSXhwg"

func TestRegistration(t *testing.T) {
	defaultStorage := new(mockDbStorage)
	defaultHandler := func() *handler {
		return &handler{defaultStorage, secret}
	}

	tests := []struct {
		name       string
		code       int
		id         string
		login      string
		password   string
		body       func(login string, password string) string
		getHandler func() *handler
	}{
		{
			name:     "register and auth success",
			code:     200,
			id:       "1",
			login:    "login",
			password: "password",
			body: func(login string, password string) string {
				return fmt.Sprintf(`{"login": "%v","password": "%v"}`, login, password)
			},
			getHandler: func() *handler {
				storage := new(mockDbStorage)
				storage.On("Register", "login", "password").Return("1", nil)
				return &handler{storage, secret}
			},
		},
		{
			name:     "bad request: incorrect login field name",
			code:     400,
			login:    "login",
			password: "password",
			body: func(login string, password string) string {
				return fmt.Sprintf(`{"login1": "%v","password": "%v"}`, login, password)
			},
			getHandler: defaultHandler,
		},
		{
			name:     "bad request: incorrect password field name",
			code:     400,
			login:    "login",
			password: "password",
			body: func(login string, password string) string {
				return fmt.Sprintf(`{"login": "%v","password2": "%v"}`, login, password)
			},
			getHandler: defaultHandler,
		},
		{
			name:     "bad request: login is empty",
			code:     400,
			login:    "",
			password: "password",
			body: func(login string, password string) string {
				return fmt.Sprintf(`{"login": "%v","password": "%v"}`, login, password)
			},
			getHandler: defaultHandler,
		},
		{
			name:     "bad request: password is empty",
			code:     400,
			login:    "login",
			password: "",
			body: func(login string, password string) string {
				return fmt.Sprintf(`{"login": "%v","password": "%v"}`, login, password)
			},
			getHandler: defaultHandler,
		},
		{
			name:     "login is already taken",
			code:     409,
			login:    "already_taken_login",
			password: "password",
			body: func(login string, password string) string {
				return fmt.Sprintf(`{"login": "%v","password": "%v"}`, login, password)
			},
			getHandler: func() *handler {
				storage := new(mockDbStorage)
				storage.On("Register", "already_taken_login", "password").Return("", db.ErrDuplicateLogin)
				return &handler{storage, secret}
			},
		},
		{
			name:     "internal server error",
			code:     500,
			login:    "internal_error_login",
			password: "password",
			body: func(login string, password string) string {
				return fmt.Sprintf(`{"login": "%v","password": "%v"}`, login, password)
			},
			getHandler: func() *handler {
				storage := new(mockDbStorage)
				storage.On("Register", "internal_error_login", "password").Return("", errors.New("unexpected exception"))
				return &handler{storage, secret}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewReader([]byte(tt.body(tt.login, tt.password))))
			request.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			h := http.HandlerFunc(tt.getHandler().Register)
			h.ServeHTTP(w, request)
			res := w.Result()
			defer res.Body.Close()
			assert.Equal(t, tt.code, res.StatusCode, "wrong status")

			if res.StatusCode == 200 {
				validateToken(t, res, tt.id, secret)
			}
		})
	}
}

func TestAuth(t *testing.T) {
	defaultStorage := new(mockDbStorage)
	defaultHandler := func() *handler {
		return &handler{defaultStorage, secret}
	}

	tests := []struct {
		name       string
		code       int
		id         string
		login      string
		password   string
		body       func(login string, password string) string
		getHandler func() *handler
	}{
		{
			name:     "auth success",
			code:     200,
			id:       "1",
			login:    "login",
			password: "password",
			body: func(login string, password string) string {
				return fmt.Sprintf(`{"login": "%v","password": "%v"}`, login, password)
			},
			getHandler: func() *handler {
				storage := new(mockDbStorage)
				storage.On("GetByLoginPassword", "login", "password").Return("1", nil)
				return &handler{storage, secret}
			},
		},
		{
			name:     "bad request: incorrect login field name",
			code:     400,
			login:    "login",
			password: "password",
			body: func(login string, password string) string {
				return fmt.Sprintf(`{"login1": "%v","password": "%v"}`, login, password)
			},
			getHandler: defaultHandler,
		},
		{
			name:     "bad request: incorrect password field name",
			code:     400,
			login:    "login",
			password: "password",
			body: func(login string, password string) string {
				return fmt.Sprintf(`{"login": "%v","password2": "%v"}`, login, password)
			},
			getHandler: defaultHandler,
		},
		{
			name:     "bad request: login is empty",
			code:     400,
			login:    "",
			password: "password",
			body: func(login string, password string) string {
				return fmt.Sprintf(`{"login": "%v","password": "%v"}`, login, password)
			},
			getHandler: defaultHandler,
		},
		{
			name:     "bad request: password is empty",
			code:     400,
			login:    "login",
			password: "",
			body: func(login string, password string) string {
				return fmt.Sprintf(`{"login": "%v","password": "%v"}`, login, password)
			},
			getHandler: defaultHandler,
		},
		{
			name:     "login incorrect",
			code:     401,
			login:    "incorrect_login",
			password: "password",
			body: func(login string, password string) string {
				return fmt.Sprintf(`{"login": "%v","password": "%v"}`, login, password)
			},
			getHandler: func() *handler {
				storage := new(mockDbStorage)
				storage.On("GetByLoginPassword", "incorrect_login", "password").Return("", db.ErrUserNotFound)
				return &handler{storage, secret}
			},
		},
		{
			name:     "password incorrect",
			code:     401,
			login:    "login",
			password: "incorrect_password",
			body: func(login string, password string) string {
				return fmt.Sprintf(`{"login": "%v","password": "%v"}`, login, password)
			},
			getHandler: func() *handler {
				storage := new(mockDbStorage)
				storage.On("GetByLoginPassword", "login", "incorrect_password").Return("", db.ErrUserNotFound)
				return &handler{storage, secret}
			},
		},
		{
			name:     "internal server error",
			code:     500,
			login:    "internal_error_login",
			password: "password",
			body: func(login string, password string) string {
				return fmt.Sprintf(`{"login": "%v","password": "%v"}`, login, password)
			},
			getHandler: func() *handler {
				storage := new(mockDbStorage)
				storage.On("GetByLoginPassword", "internal_error_login", "password").Return("", errors.New("unexpected exception"))
				return &handler{storage, secret}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewReader([]byte(tt.body(tt.login, tt.password))))
			request.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			h := http.HandlerFunc(tt.getHandler().Auth)
			h.ServeHTTP(w, request)
			res := w.Result()
			defer res.Body.Close()
			assert.Equal(t, tt.code, res.StatusCode, "wrong status")

			if res.StatusCode == 200 {
				validateToken(t, res, tt.id, secret)
			}
		})
	}
}

func validateToken(t *testing.T, res *http.Response, id string, key string) {
	cookies := res.Cookies()
	tokenFound := false
	for i := 0; i < len(cookies); i++ {
		if cookies[i].Name == "token" {
			token, err := jwt.NewWithClaims(jwt.SigningMethodHS512, UserClaims{Id: id}).SignedString([]byte(key))
			assert.NoError(t, err, "unexpected exception in validateToken")
			assert.Equal(t, "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJpZCI6IjEifQ.VsJEi0QUMf6FZ3r6p3EzRmEqbNq6sePy27Rw8nfaHDb6lyYkZdSWNGsQx6dX1dSDp3oRp8MD2fYTBJlljsjD1A", token, "bad token")
			tokenFound = true
			break
		}
	}
	if !tokenFound {
		assert.Fail(t, "token not found in cookies")
	}
}

func TestJWT(t *testing.T) {
	secret := "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJMb2dpbiI6ImxvZ2luIn0.cJ-fGT2jF6lVw1dF6MfN7k44KuNGdRowac6RXzCFO997Sjo0Uk_wNVtj2i8jtUt9_0RQI1CnsHu5dOcINSXhwg"
	token, _ := getJWTToken("1", secret)
	result, _ := GetIdFromJWTToken(token, secret)
	assert.Equal(t, "1", result)
}
