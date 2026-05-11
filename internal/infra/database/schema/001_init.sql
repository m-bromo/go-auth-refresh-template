-- +goose Up
CREATE TABLE IF NOT EXISTS users(
    id UUID PRIMARY KEY,
    email varchar(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    username TEXT UNIQUE NOT NULL
);

-- +goose Down
DROP TABLE users;