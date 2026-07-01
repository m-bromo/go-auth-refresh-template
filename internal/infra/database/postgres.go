package database

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"

	"github.com/m-bromo/go-auth-template/configs"
)

func NewPostgresConnection(postgresOptions *configs.Postgres) (*sql.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=disable",
		postgresOptions.Host,
		postgresOptions.Port,
		postgresOptions.Name,
		postgresOptions.User,
		postgresOptions.Password,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, err
}
