package order

import (
	"encoding/json"
	"errors"
	"gophermart/internal/auth"
	"gophermart/internal/db"
	"gophermart/internal/order/model/api"
	"io"
	"net/http"
	"strconv"
)

type handler struct {
	db     db.Storage
	secret string
}

func (h *handler) userId(r *http.Request) (string, bool) {
	//todo log

	if token, err := r.Cookie("token"); err != nil {
		return "", false
	} else if id, err := auth.GetIdFromJWTToken(token.Value, h.secret); err != nil {
		return "", false
	} else {
		return id, true
	}
}

func (h *handler) PostOrder(w http.ResponseWriter, r *http.Request) {
	if userId, isAuthed := h.userId(r); !isAuthed {
		w.WriteHeader(http.StatusUnauthorized)
	} else if body, err := io.ReadAll(r.Body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
	} else if order, err := strconv.Atoi(string(body)); err != nil {
		w.WriteHeader(http.StatusBadRequest)
	} else if !isValidOrder(order) {
		w.WriteHeader(http.StatusUnprocessableEntity)
	} else if err := h.db.SaveOrder(userId, order); err != nil {
		if errors.Is(err, db.ErrDuplicateOrder) {
			// 200 -  номер заказа уже был загружен этим пользователем;
			w.WriteHeader(http.StatusOK)
		} else if errors.Is(err, db.ErrOrderOfAnotherUser) {
			// 409 — номер заказа уже был загружен другим пользователем;
			w.WriteHeader(http.StatusConflict)
		} else {
			// 500 — внутренняя ошибка сервера.
			w.WriteHeader(http.StatusInternalServerError)
		}
	} else {
		// 202 -  новый номер заказа принят в обработку;
		w.WriteHeader(http.StatusAccepted)
	}
}

func (h *handler) GetOrders(w http.ResponseWriter, r *http.Request) {
	if userId, isAuthed := h.userId(r); !isAuthed {
		// 401 — пользователь не авторизован.
		w.WriteHeader(http.StatusUnauthorized)
	} else if orders, err := h.db.GetOrders(userId); err != nil {
		// 500 — внутренняя ошибка сервера.
		w.WriteHeader(http.StatusInternalServerError)
	} else if len(orders) == 0 {
		// 	204 — нет данных для ответа.
		w.WriteHeader(http.StatusNoContent)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		apiOrders := make([]api.Order, len(orders))
		for i := 0; i < len(orders); i++ {
			apiOrders[i] = orders[i].ToApi()
		}
		json.NewEncoder(w).Encode(apiOrders)
	}
}

// Valid check number is valid or not based on Luhn algorithm
func isValidOrder(number int) bool {
	return (number%10+checksum(number/10))%10 == 0
}

func checksum(number int) int {
	var luhn int

	for i := 0; number > 0; i++ {
		cur := number % 10

		if i%2 == 0 { // even
			cur = cur * 2
			if cur > 9 {
				cur = cur%10 + cur/10
			}
		}

		luhn += cur
		number = number / 10
	}
	return luhn % 10
}
