# Tech Stack — radif_service

## Language

### Go 1.23
- **Purpose:** Backend API service.
- **Why:** Fast compilation, excellent concurrency, small Docker images, great stdlib for HTTP.
- **Rules:** `cmd/api/` for binary entry point. `internal/` for all private packages. No global mutable state.
- **Docs:** https://go.dev/doc

## HTTP

### chi v5
- **Purpose:** Lightweight HTTP router and middleware stack.
- **Why:** Idiomatic net/http; composable middleware; no framework lock-in.
- **Rules:** All routes under `/api/v1/`. Protected routes wrapped with `RequireAuth` middleware.
- **Docs:** https://github.com/go-chi/chi

### go-chi/cors
- **Purpose:** CORS middleware for chi.
- **Why:** Simple, integrates natively with chi.
- **Docs:** https://github.com/go-chi/cors

## Auth

### golang-jwt/jwt/v5
- **Purpose:** JWT token creation and validation.
- **Why:** Standard, well-maintained Go JWT library.
- **Rules:** HS256 signing. 30-day expiry. Payload: `sub`, `phone`, `accountType`, `iat`, `exp`.
- **Docs:** https://github.com/golang-jwt/jwt

## Database

### PostgreSQL 16
- **Purpose:** Primary relational database.
- **Why:** Battle-tested, ACID compliant, UUID support via `gen_random_uuid()`.
- **Docs:** https://www.postgresql.org/docs

### pgx/v5
- **Purpose:** PostgreSQL driver + connection pool (`pgxpool`).
- **Why:** Best-in-class performance; native Go types for UUID, time, etc.
- **Rules:** Share a single `pgxpool.Pool` across the process. Use `pgx.ErrNoRows` for not-found detection.
- **Docs:** https://github.com/jackc/pgx

### golang-migrate/v4
- **Purpose:** Database schema migrations.
- **Why:** Supports embed.FS for shipping migrations inside the binary; up/down pairs.
- **Rules:** Migrations in `internal/db/migrations/`. Never modify an existing migration file.
- **Docs:** https://github.com/golang-migrate/migrate

## Configuration

### joho/godotenv
- **Purpose:** Load `.env` file into environment variables in development.
- **Why:** Minimal, no-magic `.env` loading.
- **Rules:** `.env` is gitignored. Use `.env.example` as template.
- **Docs:** https://github.com/joho/godotenv

## Infrastructure

### Docker (multi-stage)
- **Purpose:** Containerized build and runtime.
- **Rules:** `golang:1.23-alpine` builder, `alpine:3.20` runner. Final image has no Go toolchain.

### Docker Compose (root `docker-compose.yml`)
- **Purpose:** Orchestrates `postgres`, `api`, and `web` services together.
- **Docs:** https://docs.docker.com/compose
