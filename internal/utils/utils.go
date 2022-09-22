package utils

import (
	"math"
	"net/http"

	"github.com/golang-jwt/jwt/v4"
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
	} else if id, err := GetIdFromJWTToken(token.Value, secret); err != nil {
		return "", false
	} else {
		return id, true
	}
}

type UserClaims struct {
	Id string `json:"id"`
	jwt.StandardClaims
}

func GetJWTToken(id string, secret string) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS512, UserClaims{Id: id}).SignedString([]byte(secret))
}

func GetIdFromJWTToken(tokenString string, secret string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
		return claims.Id, nil
	} else {
		return "", err
	}
}

// Valid check number is valid or not based on Luhn algorithm
func IsValidOrder(number int) bool {
	return (number%10+checksum(number/10))%10 == 0
}

func checksum(number int) int {
	var luhn int

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
