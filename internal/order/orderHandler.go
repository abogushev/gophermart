package order

import (
	"encoding/json"
	"errors"
	"gophermart/internal/db"
	"gophermart/internal/order/model/api"
	"gophermart/internal/utils"
	"io"
	"net/http"
	"strconv"
)

type handler struct {
	db     db.Storage
	secret string
}

func (h *handler) PostOrder(w http.ResponseWriter, r *http.Request) {
	if userId, isAuthed := utils.GetUserId(r, h.secret); !isAuthed {
		w.WriteHeader(http.StatusUnauthorized)
	} else if body, err := io.ReadAll(r.Body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
	} else if order, err := strconv.Atoi(string(body)); err != nil {
		w.WriteHeader(http.StatusBadRequest)
	} else if !utils.IsValidOrder(order) {
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
	if userId, isAuthed := utils.GetUserId(r, h.secret); !isAuthed {
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
