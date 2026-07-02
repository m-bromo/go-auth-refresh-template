# Project Context

This project is a Go authentication template intended to be reused as a base for other projects.

## Purpose

- Provide a reusable authentication API foundation.
- Keep authentication concerns separated into handlers, services, repositories, infrastructure, and domain models.
- Support user registration, login, protected routes, and token refresh.

## Stack

- Go
- PostgreSQL
- sqlc
- JWT access tokens
- Refresh tokens in PostgreSQL
- OTP codes in PostgreSQL

## Authentication Flow

The authentication flow uses two token types:

- Access token: a JWT returned by the login and refresh endpoints.
- Refresh token: a UUID stored in PostgreSQL and sent to the client as an HTTP-only cookie.

General flow:

1. A user registers with email, username, and password.
2. The password is hashed before being stored in PostgreSQL.
3. A user logs in with email and password.
4. The API validates the credentials.
5. The API returns a JWT access token.
6. The API creates a refresh token, stores it in PostgreSQL, and sends it as an HTTP-only cookie.
7. Protected routes validate the JWT from the `Authorization` header.
8. The refresh endpoint reads the refresh-token cookie, rotates the refresh token, and returns a new access token.
9. OTP login and password-reset flows store short-lived OTP hashes in PostgreSQL.

## Data Storage

PostgreSQL stores persistent user data:

- user ID
- email
- hashed password
- username
- refresh token ID, user ID, creation time, and expiration time
- password reset token hashes and expiration metadata
- OTP challenge ID, identifier, hashed code, attempts, and expiration time

## Project Structure

- `main.go`: application composition root, dependency wiring, and route registration.
- `config/`: environment-based configuration.
- `internal/domain/`: domain models.
- `internal/service/`: application/business logic.
- `internal/repository/`: persistence abstractions for PostgreSQL.
- `internal/infra/database/`: PostgreSQL connection, migrations, sqlc queries, and generated sqlc code.
- `internal/web/handler/`: HTTP handlers.
- `internal/web/middleware/`: authentication middleware.
- `internal/web/cookie/`: refresh-token cookie helpers.
- `internal/pkg/`: shared validation and security helpers.
- `test/`: integration tests using testcontainers.

## Main Endpoints

- `POST /auth/register`: registers a new user.
- `POST /auth/login`: authenticates a user and issues access and refresh tokens.
- `POST /refresh`: rotates the refresh token and issues a new access token.
- `GET /user/{id}`: returns a user profile and requires JWT authentication.

## Development Notes

- Database queries are defined in SQL and generated with sqlc.
- PostgreSQL is available through `docker-compose.yml`.
- Passwords are hashed with bcrypt.
- Refresh tokens are rotated when used.
- Client-facing errors are wrapped through the API error layer.
