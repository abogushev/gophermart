package account

import (
	"encoding/json"
	"errors"
	"fmt"
	"gophermart/internal/db"
	"gophermart/internal/order/model"
	"net/http"
	"net/http/httptest"
	"testing"

	accountApi "gophermart/internal/account/model/api"
	accountModel "gophermart/internal/account/model/db"

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
	fmt.Println(args.Get(0))
	r := args.Get(0).(accountModel.Account)
	return &r, args.Error(1)
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
