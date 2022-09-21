package utils

import (
	"gophermart/internal/auth"
	"math"
	"net/http"
)

func GetApiAccrual(a int64) float64 {
	return float64(a/100) + float64(a%100)/100
}

func GetPersistentAccrual(a float64) int64 {
	integer, fraction := math.Modf(a)
	return int64(integer*100 + fraction*100)
}

func GetUserId(r *http.Request, secret string) (string, bool) {
	//todo log

	if token, err := r.Cookie("token"); err != nil {
		return "", false
	} else if id, err := auth.GetIdFromJWTToken(token.Value, secret); err != nil {
		return "", false
	} else {
		return id, true
	}
}
