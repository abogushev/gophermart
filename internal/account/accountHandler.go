package account

import (
	"encoding/json"
	"errors"
	"gophermart/internal/db"
	"gophermart/internal/utils"
	"net/http"
	"strconv"
)

type handler struct {
	db     db.Storage
	secret string
}

func (h *handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	if userId, isAuthed := utils.GetUserId(r, h.secret); !isAuthed {
		// 401 — пользователь не авторизован.
		w.WriteHeader(http.StatusUnauthorized)
	} else if account, err := h.db.GetAccount(userId); err != nil {
		if err == db.ErrUserNotFound {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(account.ToApi())
	}
}

type WithdrawData struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

func (h *handler) PostWithdraw(w http.ResponseWriter, r *http.Request) {
	var withdrawData WithdrawData
	if userId, isAuthed := utils.GetUserId(r, h.secret); !isAuthed {
		w.WriteHeader(http.StatusUnauthorized)
	} else if err := json.NewDecoder(r.Body).Decode(&withdrawData); err != nil {
		w.WriteHeader(http.StatusBadRequest)
	} else if order, err := strconv.Atoi(withdrawData.Order); err != nil || withdrawData.Sum < 0 {
		w.WriteHeader(http.StatusBadRequest)
	} else if !utils.IsValidOrder(order) {
		w.WriteHeader(http.StatusUnprocessableEntity)
	} else if err := h.db.WithdrawFromAccount(userId, withdrawData.Sum, order); err != nil {
		if errors.Is(err, db.ErrUserNotFound) {
			// 404 - account not found
			w.WriteHeader(http.StatusNotFound)
		} else if errors.Is(err, db.ErrBalanceLimitExhausted) {
			// 402 — на счету недостаточно средств
			w.WriteHeader(http.StatusPaymentRequired)
		} else {
			// 500 — внутренняя ошибка сервера.
			w.WriteHeader(http.StatusInternalServerError)
		}
	} else {
		// 202 -  новый номер заказа принят в обработку;
		w.WriteHeader(http.StatusOK)
	}
}
