package main

import (
	"log"

	"github.com/m-bromo/go-auth-template/configs"
	"github.com/m-bromo/go-auth-template/internal/infra/database"
	"github.com/pressly/goose/v3"
)

func main() {
	cfg, err := configs.NewConfig()
	if err != nil {
		log.Fatal(err.Error())
	}

	db, err := database.NewPostgresConnection(&cfg.Postgres)
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
