package config

import (
	"github.com/Netflix/go-env"
	"github.com/joho/godotenv"
)

func NewConfig() (*Config, error) {
	var config Config
	if err := godotenv.Load(); err != nil {
		return nil, err
	}

	_, err := env.UnmarshalFromEnviron(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
