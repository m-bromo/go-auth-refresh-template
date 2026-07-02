package models

import "github.com/google/uuid"

type RegisterUserPayload struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6,containsany=!@#$%&?"`
	Username string `json:"username" validate:"required,min=3,max=100"`
}

type LoginPayload struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type SendOTPPayload struct {
	Email string `json:"email" validate:"required,email"`
}

type VerifyOTPPayload struct {
	Email string `json:"email" validate:"required,email"`
	Code  string `json:"code" validate:"required"`
}

type VerifyPasswordResetCodePayload struct {
	ChallengeID uuid.UUID `json:"challenge_id" validate:"required"`
	Code        string    `json:"code" validate:"required"`
}

type ResetPasswordPayload struct {
	Password   string `json:"password" validate:"required,min=6,containsany=!@#$%&?"`
	ResetToken string `json:"reset_token" validate:"required"`
}

type LoginResponse struct {
	AccessToken string `json:"access_token"`
}

type GetProfilePayload struct {
	Email    string `json:"email"`
	Username string `json:"username"`
}

type VerifyOTPResponse struct {
	ResetToken string `json:"reset_token"`
}

type SendOTPResponse struct {
	ChallengeID uuid.UUID `json:"challenge_id"`
}
