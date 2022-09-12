package auth

import (
	"errors"

	"github.com/golang-jwt/jwt/v4"
)

type UserClaims struct {
	id string
	jwt.StandardClaims
}

func getJWTToken(id string, secret string) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS512, UserClaims{id: id}).SignedString([]byte(secret))
}

func GetIdFromJWTToken(str string, secret string) (string, error) {
	token, err := jwt.ParseWithClaims(
		str,
		&UserClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		},
	)
	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(*UserClaims)
	if !ok {
		return "", errors.New("Couldn't parse claims")
	}
	return claims.id, nil
}
