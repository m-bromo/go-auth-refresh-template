package main

import (
	"log"
	"log/slog"

	"github.com/m-bromo/go-auth-template/config"
	"github.com/m-bromo/go-auth-template/internal/app"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatal(err.Error())
	}

	app, err := app.New(cfg)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer app.DB.Close()

	slog.Info("starting application on port", "url", cfg.API.URL)

	if err := app.Server.Run(cfg.API.URL); err != nil {
		log.Fatal(err)
	}
}
