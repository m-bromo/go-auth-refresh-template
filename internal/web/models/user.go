package models

type RegisterUserPayload struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6,containsany=!@#$%&?"`
	Username string `json:"username" validate:"required,min=3,max=100"`
}

type LoginPayload struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type SendOtpLoginCodePayload struct {
	Email string `json:"email" validate:"required,email"`
}

type LoginWithOtpPayload struct {
	Email string `json:"email" validate:"required,email"`
	Code  string `json:"code" validate:"required"`
}

type LoginResponse struct {
	AccessToken string `json:"access_token"`
}

type GetProfilePayload struct {
	Email    string `json:"email"`
	Username string `json:"username"`
}
