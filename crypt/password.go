package crypt

import (
	"encoding/base64"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", nil
	}
	return base64.StdEncoding.EncodeToString(hashedPassword), nil
}
