package auth

import (
	"errors"

	"github.com/golang-jwt/jwt/v4"
)

type UserClaims struct {
	Login string
	jwt.StandardClaims
}

func getJWTToken(login string, secret string) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS512, UserClaims{Login: login}).SignedString([]byte(secret))
}

func getLoginFromJWTToken(str string, secret string) (string, error) {
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
	return claims.Login, nil
}
