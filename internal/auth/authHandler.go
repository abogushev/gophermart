package auth

import (
	"encoding/json"
	"errors"
	"gophermart/internal/db"
	"gophermart/internal/utils"
	"net/http"

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

type authData struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func parseAuthErr(authData authData, err error) string {
	if err != nil {
		return err.Error()
	}
	if authData.Login == "" {
		return "login must be non empty"
	}
	if authData.Password == "" {
		return "password must be non empty"
	}
	return ""
}

func (h *handler) Register(w http.ResponseWriter, r *http.Request) {
	var authData authData
	if err := json.NewDecoder(r.Body).Decode(&authData); err != nil || authData.Login == "" || authData.Password == "" {
		var msg string = parseAuthErr(authData, err)
		h.logger.Warnf("failed to register: %v", msg)
		http.Error(w, msg, http.StatusBadRequest)
	} else if id, err := h.db.Register(authData.Login, authData.Password); err != nil {
		if errors.Is(err, db.ErrDuplicateLogin) {
			w.WriteHeader(http.StatusConflict)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		h.logger.Warnf("failed to register: %w", err)
	} else if token, err := utils.GetJWTToken(id, h.secret); err != nil {
		h.logger.Warnf("failed to register: %w", err)
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		http.SetCookie(w, &http.Cookie{Name: "token", Value: token})
		w.WriteHeader(http.StatusOK)
	}
}

func (h *handler) Auth(w http.ResponseWriter, r *http.Request) {
	var authData authData
	if err := json.NewDecoder(r.Body).Decode(&authData); err != nil || authData.Login == "" || authData.Password == "" {
		var msg string = parseAuthErr(authData, err)
		h.logger.Warnf("failed to auth: %w", err)
		http.Error(w, msg, http.StatusBadRequest)
	} else if id, err := h.db.GetByLoginPassword(authData.Login, authData.Password); err != nil {
		if err == db.ErrUserNotFound {
			w.WriteHeader(http.StatusUnauthorized)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		h.logger.Warnf("failed to auth: %w", err)
	} else if token, err := utils.GetJWTToken(id, h.secret); err != nil {
		h.logger.Warnf("failed to auth: %w", err)
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		http.SetCookie(w, &http.Cookie{Name: "token", Value: token})
		w.WriteHeader(http.StatusOK)
	}
}
