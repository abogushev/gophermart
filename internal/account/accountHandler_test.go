package account

import (
	"bytes"
	"encoding/json"
	"errors"
	"gophermart/internal/db"
	"gophermart/internal/order/model"
	"net/http"
	"net/http/httptest"
	"testing"

	accountApi "gophermart/internal/account/model/api"
	accountModel "gophermart/internal/account/model/db"
	withdrawalsModel "gophermart/internal/withdrawals/model/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockDbStorage struct {
	mock.Mock
}

func (m *mockDbStorage) Register(login string, password string) (string, error) {
	return "", nil
}

func (m *mockDbStorage) GetByLoginPassword(login string, password string) (string, error) {
	return "", nil
}

func (m *mockDbStorage) SaveOrder(userId string, number int) error {
	args := m.Called(userId, number)
	return args.Error(0)
}

func (m *mockDbStorage) GetOrders(userId string) ([]model.Order, error) {
	args := m.Called(userId)
	return args.Get(0).([]model.Order), args.Error(1)
}

func (m *mockDbStorage) GetAccount(userId string) (*accountModel.Account, error) {
	args := m.Called(userId)
	r := args.Get(0).(accountModel.Account)
	return &r, args.Error(1)
}

func (m *mockDbStorage) WithdrawFromAccount(userId string, sum float64, number int) error {
	args := m.Called(userId, sum, number)
	r := args.Get(0)
	if r == nil {
		return nil
	}
	return r.(error)
}

func (m *mockDbStorage) GetWithdrawals(userId string) ([]withdrawalsModel.Withdrawals, error) {
	return nil, nil
}

var secret = "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJMb2dpbiI6ImxvZ2luIn0.cJ-fGT2jF6lVw1dF6MfN7k44KuNGdRowac6RXzCFO997Sjo0Uk_wNVtj2i8jtUt9_0RQI1CnsHu5dOcINSXhwg"

func Test_handler_GetAccount(t *testing.T) {
	defaultStorage := new(mockDbStorage)
	defaultHandler := func() *handler {
		return &handler{defaultStorage, secret}
	}
	defaultToken := "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJpZCI6IjEifQ.VsJEi0QUMf6FZ3r6p3EzRmEqbNq6sePy27Rw8nfaHDb6lyYkZdSWNGsQx6dX1dSDp3oRp8MD2fYTBJlljsjD1A"
	tests := []struct {
		name             string
		code             int
		token            string
		getHandler       func() *handler
		checkResponeBody func(res *http.Response)
	}{
		{
			name:  "счет найден",
			code:  200,
			token: defaultToken,
			getHandler: func() *handler {
				storage := new(mockDbStorage)

				storage.On("GetAccount", "1").Return(accountModel.Account{"1", 10, 10}, nil)
				return &handler{db: storage, secret: secret}
			},
			checkResponeBody: func(res *http.Response) {
				var result accountApi.Account
				json.NewDecoder(res.Body).Decode(&result)

				assert.Equal(t, accountApi.Account{10, 10}, result, "wrong response")
			},
		},
		{
			name:  "нет счета",
			code:  404,
			token: defaultToken,
			getHandler: func() *handler {
				storage := new(mockDbStorage)
				storage.On("GetAccount", "1").Return(accountModel.Account{}, db.ErrUserNotFound)
				return &handler{db: storage, secret: secret}
			},
			checkResponeBody: func(res *http.Response) {
				var result accountApi.Account
				json.NewDecoder(res.Body).Decode(&result)
				assert.Empty(t, result)
			},
		},
		{
			name:       "пользователь не аутентифицирован",
			code:       401,
			token:      "wrong token",
			getHandler: defaultHandler,
		},
		{
			name:  "внутренняя ошибка сервера.",
			code:  500,
			token: defaultToken,
			getHandler: func() *handler {
				storage := new(mockDbStorage)
				storage.On("GetAccount", "1").Return(accountModel.Account{}, errors.New("unexpected exception"))
				return &handler{db: storage, secret: secret}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
			request.AddCookie(&http.Cookie{Name: "token", Value: tt.token})

			w := httptest.NewRecorder()
			h := http.HandlerFunc(tt.getHandler().GetAccount)
			h.ServeHTTP(w, request)
			res := w.Result()
			defer res.Body.Close()
			if tt.checkResponeBody != nil {
				tt.checkResponeBody(res)
			}

			assert.Equal(t, tt.code, res.StatusCode, "wrong status")
		})
	}
}

func Test_handler_Withdraw(t *testing.T) {
	defaultStorage := new(mockDbStorage)
	defaultHandler := func() *handler {
		return &handler{defaultStorage, secret}
	}
	defaultBody := func() string { return `{"order": "79927398713","sum": 5.0}` }
	defaultNumber := 79927398713
	defaultToken := "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJpZCI6IjEifQ.VsJEi0QUMf6FZ3r6p3EzRmEqbNq6sePy27Rw8nfaHDb6lyYkZdSWNGsQx6dX1dSDp3oRp8MD2fYTBJlljsjD1A"
	tests := []struct {
		name       string
		code       int
		token      string
		body       func() string
		getHandler func() *handler
	}{
		{
			name: "успешная обработка запроса",
			code: 200,

			token: defaultToken,
			body:  defaultBody,
			getHandler: func() *handler {
				storage := new(mockDbStorage)
				storage.On("WithdrawFromAccount", "1", 5.0, defaultNumber).Return(nil)
				return &handler{db: storage, secret: secret}
			},
		},
		{
			name:  "на счету недостаточно средств",
			code:  402,
			token: defaultToken,
			body:  defaultBody,
			getHandler: func() *handler {
				storage := new(mockDbStorage)
				storage.On("WithdrawFromAccount", "1", 5.0, defaultNumber).Return(db.ErrBalanceLimitExhausted)
				return &handler{db: storage, secret: secret}
			},
		},
		{
			name: "неверный формат запроса",
			code: 400,

			token:      defaultToken,
			body:       func() string { return "abc" },
			getHandler: defaultHandler,
		},
		{
			name: "пользователь не аутентифицирован",
			code: 401,

			token:      "wrong token",
			body:       defaultBody,
			getHandler: defaultHandler,
		},
		{
			name: "неверный формат номера заказа",
			code: 422,

			token:      defaultToken,
			body:       func() string { return `{"order": "799273987131","sum": 5.0}` },
			getHandler: defaultHandler,
		},
		{
			name:  "внутренняя ошибка сервера.",
			code:  500,
			token: defaultToken,
			body:  defaultBody,
			getHandler: func() *handler {
				storage := new(mockDbStorage)
				storage.On("WithdrawFromAccount", "1", 5.0, defaultNumber).Return(errors.New("unexpected error"))
				return &handler{db: storage, secret: secret}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", bytes.NewReader([]byte(tt.body())))
			request.Header.Set("Content-Type", "application/json")
			request.AddCookie(&http.Cookie{Name: "token", Value: tt.token})

			w := httptest.NewRecorder()
			h := http.HandlerFunc(tt.getHandler().PostWithdraw)
			h.ServeHTTP(w, request)
			res := w.Result()
			defer res.Body.Close()
			assert.Equal(t, tt.code, res.StatusCode, "wrong status")
		})
	}
}
