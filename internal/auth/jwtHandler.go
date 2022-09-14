package auth

import (
	"github.com/golang-jwt/jwt/v4"
)

type UserClaims struct {
	Id string `json:"id"`
	jwt.StandardClaims
}

func getJWTToken(id string, secret string) (string, error) {
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
