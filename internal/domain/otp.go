package domain

import (
	"time"

	"github.com/google/uuid"
)

type OTP struct {
	ID         uuid.UUID
	Code       string
	Identifier string
	Attempts   int
	ExpiresAt  time.Time
	CreatedAt  time.Time
}
