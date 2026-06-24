package config

import "time"

type Environment string

const (
	Development Environment = "development"
	Test        Environment = "test"
	Staging     Environment = "staging"
	Production  Environment = "production"
)

type Config struct {
	Environment  Environment `env:"ENVIRONMENT,default=development"`
	API          API
	Postgres     Postgres
	Jwt          Jwt
	Redis        Redis
	RefreshToken RefreshToken
	ResetToken   ResetToken
	OTP          OTP
	Resend       Resend
}

type API struct {
	URL  string
	Host string `env:"API_HOST,default=localhost"`
	Port string `env:"API_PORT,default=8080"`
}

type Postgres struct {
	Host     string `env:"POSTGRES_HOST,default=localhost"`
	Port     string `env:"POSTGRES_PORT,default=5432"`
	Name     string `env:"POSTGRES_NAME,default=postgres_db"`
	User     string `env:"POSTGRES_USER,default=admin"`
	Password string `env:"POSTGRES_PASSWORD,default=password"`
}

type Redis struct {
	Host     string `env:"REDIS_HOST,default=localhost"`
	Port     string `env:"REDIS_PORT,default=6379"`
	Password string `env:"REDIS_PASSWORD,default="`
}

type Jwt struct {
	PrivateKey string        `env:"JWT_PRIVATE_KEY,default=change-me"`
	Duration   time.Duration `env:"JWT_DURATION,default=15m"`
}

type RefreshToken struct {
	Duration time.Duration `env:"REFRESH_TOKEN_DURATION,default=168h"`
}

type ResetToken struct {
	Secret   string        `env:"RESET_TOKEN_SECRET,default=change-me"`
	Duration time.Duration `env:"RESET_TOKEN_DURATION,default=10m"`
}

type OTP struct {
	MaxValue int           `env:"OTP_MAX_VALUE,default=1000000"`
	Secret   string        `env:"OTP_SECRET,default=change-me"`
	Duration time.Duration `env:"OTP_DURATION,default=2m"`
}

type Resend struct {
	ApiKey string `env:"RESEND_API_KEY,default="`
	Email  string `env:"RESEND_EMAIL,default=example@email.com"`
}

func (c *Config) IsDevelopment() bool {
	return c.Environment == Development
}

func (c *Config) IsTesting() bool {
	return c.Environment == Test
}

func (c *Config) IsStaging() bool {
	return c.Environment == Staging
}

func (c *Config) IsProduction() bool {
	return c.Environment == Production
}
