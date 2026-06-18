package secure

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
)

func HashOTP(code string, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(code))

	return hex.EncodeToString(mac.Sum(nil))
}

func VerifyOTP(code string, hashedCode string, secret []byte) bool {
	hash := HashOTP(code, secret)

	return subtle.ConstantTimeCompare(
		[]byte(hash),
		[]byte(hashedCode),
	) == 1
}
