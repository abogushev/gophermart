package withdrawals

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	accountModel "gophermart/internal/account/model/db"
	"gophermart/internal/order/model"
	"gophermart/internal/withdrawals/model/api"
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
	return nil, nil
}
func (m *mockDbStorage) WithdrawFromAccount(userId string, sum float64, number int) error {
	return nil
}
func (m *mockDbStorage) GetWithdrawals(userId string) ([]withdrawalsModel.Withdrawals, error) {
	args := m.Called(userId)
	return args.Get(0).([]withdrawalsModel.Withdrawals), args.Error(1)
}

var secret = "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJMb2dpbiI6ImxvZ2luIn0.cJ-fGT2jF6lVw1dF6MfN7k44KuNGdRowac6RXzCFO997Sjo0Uk_wNVtj2i8jtUt9_0RQI1CnsHu5dOcINSXhwg"

func Test_handler_GetWithdrawals(t *testing.T) {
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
			name:  "успешная обработка запроса",
			code:  200,
			token: defaultToken,
			getHandler: func() *handler {
				storage := new(mockDbStorage)
				result := make([]withdrawalsModel.Withdrawals, 1)
				result[0] = *withdrawalsModel.NewWithdrawals("1", 10, 9278923470)
				result[0].ProcessedAt, _ = time.Parse(time.RFC3339, "2020-12-10T15:15:45+03:00")

				storage.On("GetWithdrawals", "1").Return(result, nil)
				return &handler{db: storage, secret: secret}
			},
			checkResponeBody: func(res *http.Response) {
				processedAt, _ := time.Parse(time.RFC3339, "2020-12-10T15:15:45+03:00")
				expected := make([]api.Withdrawals, 1)
				expected[0] = api.Withdrawals{"9278923470", 10, processedAt}

				var result []api.Withdrawals
				json.NewDecoder(res.Body).Decode(&result)
				assert.ElementsMatch(t, result, expected, "wrong response")
			},
		},
		{
			name:  "нет данных для ответа",
			code:  204,
			token: defaultToken,
			getHandler: func() *handler {
				storage := new(mockDbStorage)
				storage.On("GetWithdrawals", "1").Return([]withdrawalsModel.Withdrawals{}, nil)
				return &handler{db: storage, secret: secret}
			},
			checkResponeBody: func(res *http.Response) {
				var result []api.Withdrawals
				e := json.NewDecoder(res.Body).Decode(&result)
				assert.ErrorIs(t, e, io.EOF)
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
				storage.On("GetWithdrawals", "1").Return([]withdrawalsModel.Withdrawals{}, errors.New("unexpected exception"))
				return &handler{db: storage, secret: secret}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil)
			request.AddCookie(&http.Cookie{Name: "token", Value: tt.token})

			w := httptest.NewRecorder()
			h := http.HandlerFunc(tt.getHandler().GetWithdrawals)
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
