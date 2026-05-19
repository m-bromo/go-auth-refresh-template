package config

import "time"

type Config struct {
	Environment  string `env:"ENVIRONMENT,default=development"`
	API          API
	Postgres     Postgres
	Jwt          Jwt
	Redis        Redis
	RefreshToken RefreshToken
}

type API struct {
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
	Password string `env:"REDISS_PASSWORD"`
}

type Jwt struct {
	PrivateKey string        `env:"JWT_PRIVATE_KEY"`
	PublicKey  string        `env:"JWT_PUBLIC_KEY"`
	Duration   time.Duration `env:"JWT_DURATION,default=15m"`
}

type RefreshToken struct {
	Duration time.Duration `env:"REFRESH_TOKEN_DURATION,default=168h"`
}
