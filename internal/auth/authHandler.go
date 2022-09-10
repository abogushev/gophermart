package auth

import (
	"encoding/json"
	"errors"
	"net/http"
)

type Storage interface {
	Register(login, password string) error
	Find(login, password string) error
}

type handler struct {
	db     Storage
	secret string
}

///

var ErrDuplicateLogin = errors.New("login already exist")
var ErrLoginPwdIncorrect = errors.New("login/password incorrect")

///

type AuthData struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (h *handler) Register(w http.ResponseWriter, r *http.Request) {
	var authData AuthData
	if err := json.NewDecoder(r.Body).Decode(&authData); err != nil || authData.Login == "" || authData.Password == "" {
		var msg string
		if err != nil {
			msg = err.Error()
		} else if authData.Login == "" {
			msg = "login must be non empty"
		} else if authData.Password == "" {
			msg = "password must be non empty"
		}
		http.Error(w, msg, http.StatusBadRequest)
	} else if err := h.db.Register(authData.Login, authData.Password); err != nil {
		if errors.Is(err, ErrDuplicateLogin) {
			w.WriteHeader(http.StatusConflict)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	} else if token, err := getJWTToken(authData.Login, h.secret); err != nil {
		//todo add logger
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		http.SetCookie(w, &http.Cookie{Name: "token", Value: token})
		w.WriteHeader(http.StatusOK)
	}
}

func (h *handler) Auth(w http.ResponseWriter, r *http.Request) {
	var authData AuthData
	if err := json.NewDecoder(r.Body).Decode(&authData); err != nil || authData.Login == "" || authData.Password == "" {
		var msg string
		if err != nil {
			msg = err.Error()
		} else if authData.Login == "" {
			msg = "login must be non empty"
		} else if authData.Password == "" {
			msg = "password must be non empty"
		}
		http.Error(w, msg, http.StatusBadRequest)
	} else if err := h.db.Find(authData.Login, authData.Password); err != nil {
		if errors.Is(err, ErrLoginPwdIncorrect) {
			w.WriteHeader(http.StatusUnauthorized)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	} else if token, err := getJWTToken(authData.Login, h.secret); err != nil {
		//todo add logger
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		http.SetCookie(w, &http.Cookie{Name: "token", Value: token})
		w.WriteHeader(http.StatusOK)
	}
}
