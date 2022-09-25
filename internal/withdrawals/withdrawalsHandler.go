package withdrawals

import (
	"encoding/json"
	"gophermart/internal/db"
	"gophermart/internal/utils"
	"gophermart/internal/withdrawals/model/api"
	"net/http"
)

type handler struct {
	db     db.Storage
	secret string
}

func (h *handler) GetWithdrawals(w http.ResponseWriter, r *http.Request) {
	if UserID, isAuthed := utils.GetUserID(r, h.secret); !isAuthed {
		// 401 — пользователь не авторизован.
		w.WriteHeader(http.StatusUnauthorized)
	} else if withdrawals, err := h.db.GetWithdrawals(UserID); err != nil {
		// 500 — внутренняя ошибка сервера.
		w.WriteHeader(http.StatusInternalServerError)
	} else if len(withdrawals) == 0 {
		// 	204 — нет данных для ответа.
		w.WriteHeader(http.StatusNoContent)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		apiWithdrawals := make([]api.Withdrawals, len(withdrawals))
		for i := 0; i < len(apiWithdrawals); i++ {
			// apiWithdrawals[i] = withdrawals[i].ToAPI()
		}
		json.NewEncoder(w).Encode(apiWithdrawals)
	}
}
