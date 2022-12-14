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
	"gophermart/internal/db"
	"gophermart/internal/order/model"
	"gophermart/internal/utils"
	"gophermart/internal/withdrawals/model/api"
	withdrawalsModel "gophermart/internal/withdrawals/model/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockDBStorage struct {
	mock.Mock
}

func (m *mockDBStorage) Register(login string, password string) (string, error) {
	return "", nil
}

func (m *mockDBStorage) GetByLoginPassword(login string, password string) (string, error) {
	return "", nil
}

func (m *mockDBStorage) SaveOrder(UserID string, number uint64) error {
	args := m.Called(UserID, number)
	return args.Error(0)
}

func (m *mockDBStorage) GetOrders(UserID string) ([]model.Order, error) {
	args := m.Called(UserID)
	return args.Get(0).([]model.Order), args.Error(1)
}

func (m *mockDBStorage) GetAccount(UserID string) (*accountModel.Account, error) {
	return nil, nil
}
func (m *mockDBStorage) WithdrawFromAccount(UserID string, sum float64, number uint64) error {
	return nil
}
func (m *mockDBStorage) GetWithdrawals(UserID string) ([]withdrawalsModel.Withdrawals, error) {
	args := m.Called(UserID)
	return args.Get(0).([]withdrawalsModel.Withdrawals), args.Error(1)
}
func (m *mockDBStorage) CalcAmounts(offset, limit int,
	updF func(nums []int64) map[int64]db.CalcAmountsUpdateResult) (int, error) {
	return 0, nil
}

func Test_handler_GetWithdrawals(t *testing.T) {
	defaultStorage := new(mockDBStorage)
	defaultHandler := func() *handler {
		return &handler{defaultStorage, utils.TestSecret}
	}
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
			token: utils.TestToken,
			getHandler: func() *handler {
				storage := new(mockDBStorage)
				result := make([]withdrawalsModel.Withdrawals, 1)
				result[0] = *withdrawalsModel.NewWithdrawals("1", 10, 9278923470)
				result[0].ProcessedAt, _ = time.Parse(time.RFC3339, "2020-12-10T15:15:45+03:00")

				storage.On("GetWithdrawals", "1").Return(result, nil)
				return &handler{db: storage, secret: utils.TestSecret}
			},
			checkResponeBody: func(res *http.Response) {
				processedAt, _ := time.Parse(time.RFC3339, "2020-12-10T15:15:45+03:00")
				expected := make([]api.Withdrawals, 1)
				expected[0] = api.Withdrawals{Order: "9278923470", Sum: 10, ProcessedAt: processedAt}

				var result []api.Withdrawals
				json.NewDecoder(res.Body).Decode(&result)
				assert.ElementsMatch(t, result, expected, "wrong response")
			},
		},
		{
			name:  "нет данных для ответа",
			code:  204,
			token: utils.TestToken,
			getHandler: func() *handler {
				storage := new(mockDBStorage)
				storage.On("GetWithdrawals", "1").Return([]withdrawalsModel.Withdrawals{}, nil)
				return &handler{db: storage, secret: utils.TestSecret}
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
			token: utils.TestToken,
			getHandler: func() *handler {
				storage := new(mockDBStorage)
				storage.On("GetWithdrawals", "1").Return([]withdrawalsModel.Withdrawals{}, errors.New("unexpected exception"))
				return &handler{db: storage, secret: utils.TestSecret}
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
