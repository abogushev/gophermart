package account

import (
	"encoding/json"
	"gophermart/internal/db"
	"gophermart/internal/utils"
	"net/http"
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
