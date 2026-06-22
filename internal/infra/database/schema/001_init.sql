-- +goose Up
CREATE TABLE IF NOT EXISTS users(
    id UUID PRIMARY KEY,
    email varchar(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    username TEXT UNIQUE NOT NULL
);
CREATE TABLE IF NOT EXISTS refresh_tokens(
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE IF NOT EXISTS password_reset_tokens(
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT UNIQUE NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE refresh_tokens;
DROP TABLE password_reset_tokens;
DROP TABLE users;