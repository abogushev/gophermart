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

	"go.uber.org/zap"
)

type handler struct {
	db     db.Storage
	secret string
	logger *zap.SugaredLogger
}

func NewHandler(db db.Storage, secret string, logger *zap.SugaredLogger) *handler {
	return &handler{db, secret, logger}
}

func (h *handler) PostOrder(w http.ResponseWriter, r *http.Request) {
	if UserID, isAuthed := utils.GetUserID(r, h.secret); !isAuthed {
		h.logger.Warnf("failed to auth")
		w.WriteHeader(http.StatusUnauthorized)
	} else if body, err := io.ReadAll(r.Body); err != nil {
		h.logger.Warnf("failed to PostOrder: %w", err)
		w.WriteHeader(http.StatusBadRequest)
	} else if order, err := strconv.ParseUint(string(body), 10, 64); err != nil {
		h.logger.Warnf("failed to PostOrder: %w", err)
		w.WriteHeader(http.StatusBadRequest)
	} else if !utils.IsValidOrder(order) {
		h.logger.Warnf("failed to PostOrder: invalid order number")
		w.WriteHeader(http.StatusUnprocessableEntity)
	} else if err := h.db.SaveOrder(UserID, order); err != nil {
		if errors.Is(err, db.ErrDuplicateOrder) {
			// 200 -  номер заказа уже был загружен этим пользователем;
			h.logger.Warnf("failed to PostOrder: %w", err)
			w.WriteHeader(http.StatusOK)
		} else if errors.Is(err, db.ErrOrderOfAnotherUser) {
			// 409 — номер заказа уже был загружен другим пользователем;
			h.logger.Warnf("failed to PostOrder: %w", err)
			w.WriteHeader(http.StatusConflict)
		} else {
			h.logger.Errorf("failed to PostOrder: %w", err)
			// 500 — внутренняя ошибка сервера.
			w.WriteHeader(http.StatusInternalServerError)
		}
	} else {
		// 202 -  новый номер заказа принят в обработку;
		w.WriteHeader(http.StatusAccepted)
	}
}

func (h *handler) GetOrders(w http.ResponseWriter, r *http.Request) {
	if UserID, isAuthed := utils.GetUserID(r, h.secret); !isAuthed {
		// 401 — пользователь не авторизован.
		h.logger.Warnf("failed to auth")
		w.WriteHeader(http.StatusUnauthorized)
	} else if orders, err := h.db.GetOrders(UserID); err != nil {
		// 500 — внутренняя ошибка сервера.
		h.logger.Errorf("failed to GetOrders: %w", err)
		w.WriteHeader(http.StatusInternalServerError)
	} else if len(orders) == 0 {
		// 	204 — нет данных для ответа
		w.WriteHeader(http.StatusNoContent)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		apiOrders := make([]api.Order, len(orders))
		for i := 0; i < len(orders); i++ {
			apiOrders[i] = orders[i].ToAPI()
		}
		json.NewEncoder(w).Encode(apiOrders)
	}
}
