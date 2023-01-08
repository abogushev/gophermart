package utils

import (
	"errors"
	"math"
	"net/http"

	"github.com/golang-jwt/jwt/v4"
)

const (
	TestToken  = "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJpZCI6IjEifQ.VsJEi0QUMf6FZ3r6p3EzRmEqbNq6sePy27Rw8nfaHDb6lyYkZdSWNGsQx6dX1dSDp3oRp8MD2fYTBJlljsjD1A"
	TestSecret = "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJMb2dpbiI6ImxvZ2luIn0.cJ-fGT2jF6lVw1dF6MfN7k44KuNGdRowac6RXzCFO997Sjo0Uk_wNVtj2i8jtUt9_0RQI1CnsHu5dOcINSXhwg"
)

func GetAPIAccrual(a int64) float64 {
	return float64(a/100) + float64(a%100)/100
}

func GetPersistentAccrual(a float64) int64 {
	integer, fraction := math.Modf(a)
	return int64(integer*100 + fraction*100)
}

func GetUserID(r *http.Request, secret string) (string, bool) {
	if token, err := r.Cookie("token"); err != nil {
		return "", false
	} else if id, err := GetIDFromJWTToken(token.Value, secret); err != nil {
		return "", false
	} else {
		return id, true
	}
}

type UserClaims struct {
	ID string `json:"id"`
	jwt.StandardClaims
}

func GetJWTToken(id string, secret string) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS512, UserClaims{ID: id}).SignedString([]byte(secret))
}

func GetIDFromJWTToken(tokenString string, secret string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
		return claims.ID, nil
	}
	return "", errors.New("bad token")
}

// Valid check number is valid or not based on Luhn algorithm
func IsValidOrder(number uint64) bool {
	return (number%10+checksum(number/10))%10 == 0
}

func checksum(number uint64) uint64 {
	var luhn uint64

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
