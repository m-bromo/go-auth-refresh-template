package secure

import (
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hashedPassword), err
}

func CheckPassword(savedPassword string, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(savedPassword), []byte(password))
	return err == nil
}
