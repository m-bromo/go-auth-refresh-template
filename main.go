package main

import (
	"fmt"
	"log"
	"log/slog"

	"github.com/m-bromo/go-auth-template/config"
	"github.com/m-bromo/go-auth-template/internal/app"
)

func main() {
	slog.Info("starting application")

	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatal(err.Error())
	}

	app, err := app.New(cfg)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer app.DB.Close()

	if err := app.Server.Run(fmt.Sprintf("%s:%s", cfg.API.Host, cfg.API.Port)); err != nil {
		log.Fatal(err)
	}
}
