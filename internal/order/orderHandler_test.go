package order

import (
	"bytes"
	"encoding/json"
	"errors"
	"gophermart/internal/db"
	"gophermart/internal/order/model"
	"gophermart/internal/order/model/api"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	accountModel "gophermart/internal/account/model/db"
	withdrawalsModel "gophermart/internal/withdrawals/model/db"
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

func (m *mockDBStorage) SaveOrder(UserID string, number int) error {
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
func (m *mockDBStorage) WithdrawFromAccount(UserID string, sum float64, number int) error {
	return nil
}

func (m *mockDBStorage) GetWithdrawals(UserID string) ([]withdrawalsModel.Withdrawals, error) {
	return nil, nil
}

func (m *mockDBStorage) CalcAmounts(offset, limit int,
	updF func(nums []int64) map[int64]db.CalcAmountsUpdateResult) (int, error) {
	return 0, nil
}

var secret = "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJMb2dpbiI6ImxvZ2luIn0.cJ-fGT2jF6lVw1dF6MfN7k44KuNGdRowac6RXzCFO997Sjo0Uk_wNVtj2i8jtUt9_0RQI1CnsHu5dOcINSXhwg"
var logger = zap.NewExample().Sugar()

func Test_handler_PostOrder(t *testing.T) {
	defaultStorage := new(mockDBStorage)
	defaultHandler := func() *handler {
		return &handler{defaultStorage, secret, logger}
	}
	defaultBody := func(number int) string { return strconv.Itoa(number) }
	defaultNumber := 79927398713
	defaultToken := "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJpZCI6IjEifQ.VsJEi0QUMf6FZ3r6p3EzRmEqbNq6sePy27Rw8nfaHDb6lyYkZdSWNGsQx6dX1dSDp3oRp8MD2fYTBJlljsjD1A"
	tests := []struct {
		name       string
		code       int
		token      string
		number     int
		body       func(number int) string
		getHandler func() *handler
	}{
		{
			name:   "номер заказа уже был загружен этим пользователем",
			code:   200,
			number: defaultNumber,
			token:  defaultToken,
			body:   defaultBody,
			getHandler: func() *handler {
				storage := new(mockDBStorage)
				storage.On("SaveOrder", "1", defaultNumber).Return(db.ErrDuplicateOrder)
				return &handler{db: storage, secret: secret, logger: logger}
			},
		},
		{
			name:   " новый номер заказа принят в обработку",
			code:   202,
			number: defaultNumber,
			token:  defaultToken,
			body:   defaultBody,
			getHandler: func() *handler {
				storage := new(mockDBStorage)
				storage.On("SaveOrder", "1", defaultNumber).Return(nil)
				return &handler{db: storage, secret: secret, logger: logger}
			},
		},
		{
			name:       "неверный формат запроса",
			code:       400,
			number:     defaultNumber,
			token:      defaultToken,
			body:       func(number int) string { return "abc" },
			getHandler: defaultHandler,
		},
		{
			name:       "пользователь не аутентифицирован",
			code:       401,
			number:     defaultNumber,
			token:      "wrong token",
			body:       defaultBody,
			getHandler: defaultHandler,
		},
		{
			name:   "номер заказа уже был загружен другим пользователем",
			code:   409,
			number: defaultNumber,
			token:  defaultToken,
			body:   defaultBody,
			getHandler: func() *handler {
				storage := new(mockDBStorage)
				storage.On("SaveOrder", "1", defaultNumber).Return(db.ErrOrderOfAnotherUser)
				return &handler{db: storage, secret: secret, logger: logger}
			},
		},
		{
			name:       "неверный формат номера заказа",
			code:       422,
			number:     defaultNumber,
			token:      defaultToken,
			body:       func(number int) string { return strconv.Itoa(defaultNumber + 1) },
			getHandler: defaultHandler,
		},
		{
			name:   "внутренняя ошибка сервера.",
			code:   500,
			number: defaultNumber,
			token:  defaultToken,
			body:   defaultBody,
			getHandler: func() *handler {
				storage := new(mockDBStorage)
				storage.On("SaveOrder", "1", defaultNumber).Return(errors.New("unexpected exception"))
				return &handler{db: storage, secret: secret, logger: logger}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewReader([]byte(tt.body(tt.number))))
			request.Header.Set("Content-Type", "text/plain")
			request.AddCookie(&http.Cookie{Name: "token", Value: tt.token})

			w := httptest.NewRecorder()
			h := http.HandlerFunc(tt.getHandler().PostOrder)
			h.ServeHTTP(w, request)
			res := w.Result()
			defer res.Body.Close()
			assert.Equal(t, tt.code, res.StatusCode, "wrong status")
		})
	}
}

func Test_handler_GetOrders(t *testing.T) {
	defaultStorage := new(mockDBStorage)
	defaultHandler := func() *handler {
		return &handler{defaultStorage, secret, logger}
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
				storage := new(mockDBStorage)
				result := make([]model.Order, 4)
				result[0] = *model.NewOrder(9278923470, "1", model.Processed, 500.0)
				result[1] = *model.NewOrder(12345678903, "1", model.Processing, 0)
				result[2] = *model.NewOrder(346436439, "1", model.Invalid, 0)
				result[3] = *model.NewOrder(346436431, "1", model.New, 0)
				for i := 0; i < len(result); i++ {
					result[i].UploadedAt, _ = time.Parse(time.RFC3339, "2020-12-10T15:15:45+03:00")
				}

				storage.On("GetOrders", "1").Return(result, nil)
				return &handler{db: storage, secret: secret, logger: logger}
			},
			checkResponeBody: func(res *http.Response) {
				uploadedAt, _ := time.Parse(time.RFC3339, "2020-12-10T15:15:45+03:00")
				expected := make([]api.Order, 4)
				prcsAcc := 500.0

				expected[0] = api.Order{Number: "9278923470", UserID: "1", Status: "PROCESSED", UploadedAt: uploadedAt, Accrual: &prcsAcc}
				expected[1] = api.Order{Number: "12345678903", UserID: "1", Status: "PROCESSING", UploadedAt: uploadedAt, Accrual: nil}
				expected[2] = api.Order{Number: "346436439", UserID: "1", Status: "INVALID", UploadedAt: uploadedAt, Accrual: nil}
				expected[3] = api.Order{Number: "346436431", UserID: "1", Status: "NEW", UploadedAt: uploadedAt, Accrual: nil}

				var result []api.Order
				json.NewDecoder(res.Body).Decode(&result)
				assert.ElementsMatch(t, result, expected, "wrong response")
			},
		},
		{
			name:  "нет данных для ответа",
			code:  204,
			token: defaultToken,
			getHandler: func() *handler {
				storage := new(mockDBStorage)
				storage.On("GetOrders", "1").Return(make([]model.Order, 0), nil)
				return &handler{db: storage, secret: secret, logger: logger}
			},
			checkResponeBody: func(res *http.Response) {
				var result []api.Order
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
				storage := new(mockDBStorage)
				storage.On("GetOrders", "1").Return([]model.Order{}, errors.New("unexpected exception"))
				return &handler{db: storage, secret: secret, logger: logger}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
			request.AddCookie(&http.Cookie{Name: "token", Value: tt.token})

			w := httptest.NewRecorder()
			h := http.HandlerFunc(tt.getHandler().GetOrders)
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
