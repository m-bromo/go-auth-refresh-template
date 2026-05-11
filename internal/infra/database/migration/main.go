package main

import (
	"log"

	"github.com/m-bromo/go-auth-template/config"
	"github.com/m-bromo/go-auth-template/internal/infra/database"
	"github.com/pressly/goose/v3"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatal(err.Error())
	}

	db, err := database.NewPostgresConnection(cfg)
	if err != nil {
		log.Fatal(err.Error())
	}

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatal(err.Error())
	}

	if err := goose.Up(db, "internal/infra/database/schema"); err != nil {
		log.Fatal(err.Error())
	}
}
