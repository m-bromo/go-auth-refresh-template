package secure

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
)

const ResetTokenBytes = 32

func GenerateResetToken() (string, error) {
	token := make([]byte, ResetTokenBytes)
	if _, err := rand.Read(token); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(token), nil
}

func HashResetToken(token string, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(token))

	return hex.EncodeToString(mac.Sum(nil))
}

func VerifyResetToken(code string, hashedCode string, secret []byte) bool {
	hash := HashResetToken(code, secret)

	return subtle.ConstantTimeCompare(
		[]byte(hash),
		[]byte(hashedCode),
	) == 1
}
