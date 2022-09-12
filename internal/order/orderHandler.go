package order

import (
	"gophermart/internal/auth"
	"gophermart/internal/db"
	"io"
	"net/http"
	"strconv"
)

type handler struct {
	db     db.Storage
	secret string
}

func (h *handler) userIsAuthed(r *http.Request) (string, bool) {
	cookies := r.Cookies()
	for i := 0; i < len(cookies); i++ {
		if cookies[i].Name == "token" {
			if login, err := auth.GetIdFromJWTToken(cookies[i].Value, h.secret); err != nil {
				//todo log
				return "", false
			} else {
				return login, true
			}
		}
	}
}

func (h *handler) PostOrder(w http.ResponseWriter, r *http.Request) {
	if login, isAuthed := h.userIsAuthed(r); !isAuthed {
		w.WriteHeader(http.StatusUnauthorized)
	} else if body, err := io.ReadAll(r.Body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
	} else if order, err := strconv.Atoi(string(body)); err != nil {
		w.WriteHeader(http.StatusBadRequest)
	} else if !isValidOrder(order) {
		w.WriteHeader(http.StatusUnprocessableEntity)
	} else if err := h.db.saveOrder(login, order); err != nil {
		// 200 -  номер заказа уже был загружен этим пользователем;
		// 409 — номер заказа уже был загружен другим пользователем;
		// 500 — внутренняя ошибка сервера.
	} else {
		// 202
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
