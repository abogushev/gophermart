package main

import (
	"fmt"

	"github.com/golang-jwt/jwt"
)

type UserClaims struct {
	Login string
	jwt.StandardClaims
}

func main() {
	fmt.Println(jwt.NewWithClaims(jwt.SigningMethodHS512, UserClaims{Login: "login"}).SignedString([]byte("e779d710cc6fb074deedff4a5540640acd82ad45ce4abad8f33a68f27d81f0c5")))
}
