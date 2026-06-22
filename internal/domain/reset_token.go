package domain

import (
	"time"

	"github.com/google/uuid"
)

type ResetToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	UsedAt    time.Time
	CreatedAt time.Time
}
