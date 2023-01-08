package account

import (
	"encoding/json"
	"errors"
	"gophermart/internal/db"
	"gophermart/internal/utils"
	"net/http"
	"strconv"

	"go.uber.org/zap"
)

type handler struct {
	db     db.Storage
	secret string
	logger *zap.SugaredLogger
}

func NewAccountHandler(db db.Storage, secret string, logger *zap.SugaredLogger) *handler {
	return &handler{db, secret, logger}
}

func (h *handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	if UserID, isAuthed := utils.GetUserID(r, h.secret); !isAuthed {
		// 401 — пользователь не авторизован.
		h.logger.Warn("failed to auth user")
		w.WriteHeader(http.StatusUnauthorized)
	} else if account, err := h.db.GetAccount(UserID); err != nil {
		if err == db.ErrUserNotFound {
			w.WriteHeader(http.StatusNoContent)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}

		h.logger.Warnf("failed to auth: %w", err)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(account.ToAPI())
	}
}

type WithdrawData struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

func (h *handler) PostWithdraw(w http.ResponseWriter, r *http.Request) {
	var withdrawData WithdrawData
	if UserID, isAuthed := utils.GetUserID(r, h.secret); !isAuthed {
		w.WriteHeader(http.StatusUnauthorized)
		h.logger.Warn("failed to auth user")
	} else if err := json.NewDecoder(r.Body).Decode(&withdrawData); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.logger.Warnf("failed to PostWithdraw: %w", err)
	} else if order, err := strconv.ParseUint(withdrawData.Order, 10, 64); err != nil || withdrawData.Sum < 0 {
		w.WriteHeader(http.StatusBadRequest)
		h.logger.Warnf("failed to PostWithdraw: %w", err)
	} else if !utils.IsValidOrder(order) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		h.logger.Warnf("failed to PostWithdraw: invalid order")
	} else if err := h.db.WithdrawFromAccount(UserID, withdrawData.Sum, order); err != nil {
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
		h.logger.Warnf("failed to PostWithdraw: %w", err)
	} else {
		// 202 -  новый номер заказа принят в обработку;
		w.WriteHeader(http.StatusOK)
	}
}
