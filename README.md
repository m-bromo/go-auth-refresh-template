# Go Auth Refresh Template

A reusable Go authentication API template with registration, login, JWT access tokens, and Redis-backed refresh-token rotation.

The project is structured around clear HTTP, service, repository, infrastructure, and domain layers so it can be used as a starting point for authenticated Go APIs.

## Getting Started

### Prerequisites

- Go 1.26.2 or newer
- Docker and Docker Compose
- PostgreSQL and Redis, either through Docker Compose or your own local services

### Configuration

Create a `.env` file in the repository root:

```env
ENVIRONMENT=development
API_HOST=localhost
API_PORT=8080

POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_NAME=postgres_db
POSTGRES_USER=admin
POSTGRES_PASSWORD=password

REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

JWT_PRIVATE_KEY=change-me
JWT_DURATION=15m
REFRESH_TOKEN_DURATION=168h
```

### Run Locally

Start PostgreSQL and Redis:

```bash
make docker-up
```

Run database migrations:

```bash
make migrate
```

Start the API:

```bash
make run
```

The API listens on `http://localhost:8080` by default.

### Build

```bash
make build
```

### Test

```bash
make test
```

The current test suite uses testcontainers for PostgreSQL and Redis, so Docker must be available when running tests.

## API

### Register

```http
POST /auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "username": "newUser",
  "password": "password@123"
}
```

Returns `201 Created` when the user is registered.

### Login

```http
POST /auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password@123"
}
```

Returns an access token in the response body and stores the refresh token in an HTTP-only cookie.

```json
{
  "access_token": "<jwt>"
}
```

### Refresh Token

```http
POST /refresh
Cookie: auth_cookie=<refresh-token>
```

Consumes the current refresh token, stores a new refresh token cookie, and returns a new access token.

### Get User Profile

```http
GET /user/{id}
Authorization: Bearer <jwt>
```

Returns the authenticated user's profile:

```json
{
  "email": "user@example.com",
  "username": "newUser"
}
```

## Features

- User registration with email, username, and password validation
- Password hashing with bcrypt
- Login with JWT access-token generation
- Refresh tokens stored in Redis with expiration
- Refresh-token rotation on use
- HTTP-only refresh-token cookie handling
- Protected route middleware using the `Authorization: Bearer <jwt>` header
- PostgreSQL persistence generated through sqlc
- Database migrations with goose
- Integration tests backed by testcontainers

## Project Structure

```text
.
|-- config/                         Environment-based configuration
|-- internal/client_errors/          Client-facing API error helpers
|-- internal/domain/                 Domain models
|-- internal/infra/cache/            Redis client setup
|-- internal/infra/database/         PostgreSQL connection, schema, queries, migrations, sqlc output
|-- internal/pkg/                    Shared validation and security helpers
|-- internal/repository/             Persistence interfaces and implementations
|-- internal/service/                Authentication, JWT, refresh-token, and user services
|-- internal/web/                    HTTP server, routes, handlers, middleware, cookies, and models
|-- test/                            Integration tests
|-- docker-compose.yml               Local PostgreSQL and Redis services
|-- makefile                         Common development commands
`-- main.go                          Application entrypoint and dependency wiring
```

## Development

Common commands:

```bash
make docker-up     # Start PostgreSQL and Redis
make migrate       # Apply database migrations
make run           # Run the API
make build         # Build the binary
make test          # Run tests
make docker-down   # Stop and remove local containers and volumes
```

Regenerate sqlc code after changing SQL files:

```bash
sqlc generate
```

## Contributing

This repository does not currently include a separate `CONTRIBUTING.md`. For small changes, keep the existing package structure, run `gofmt`, and run `make test` before opening a pull request.

## License

No license file is currently included.
