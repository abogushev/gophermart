package order

import (
	"bytes"
	"errors"
	"gophermart/internal/db"
	"gophermart/internal/order/model"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

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

var secret = "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJMb2dpbiI6ImxvZ2luIn0.cJ-fGT2jF6lVw1dF6MfN7k44KuNGdRowac6RXzCFO997Sjo0Uk_wNVtj2i8jtUt9_0RQI1CnsHu5dOcINSXhwg"

func Test_handler_PostOrder(t *testing.T) {
	defaultStorage := new(mockDbStorage)
	defaultHandler := func() *handler {
		return &handler{defaultStorage, secret}
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
				storage := new(mockDbStorage)
				storage.On("SaveOrder", "1", defaultNumber).Return(db.ErrDuplicateOrder)
				return &handler{db: storage, secret: secret}
			},
		},
		{
			name:   " новый номер заказа принят в обработку",
			code:   202,
			number: defaultNumber,
			token:  defaultToken,
			body:   defaultBody,
			getHandler: func() *handler {
				storage := new(mockDbStorage)
				storage.On("SaveOrder", "1", defaultNumber).Return(nil)
				return &handler{db: storage, secret: secret}
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
				storage := new(mockDbStorage)
				storage.On("SaveOrder", "1", defaultNumber).Return(db.ErrOrderOfAnotherUser)
				return &handler{db: storage, secret: secret}
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
				storage := new(mockDbStorage)
				storage.On("SaveOrder", "1", defaultNumber).Return(errors.New("unexpected exception"))
				return &handler{db: storage, secret: secret}
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
