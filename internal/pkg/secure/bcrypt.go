package secure

import (
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hashedPassword), err
}

func CheckPassword(password string, savedPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(password), []byte(savedPassword))
	return err == nil
}
