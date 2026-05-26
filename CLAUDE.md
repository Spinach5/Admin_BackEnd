# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

```bash
# Copy and configure environment
cp .env.example .env

# Run database migration (creates tables + default admin user)
go run ./cmd/migrate

# Start the server (default :3001)
go run ./cmd/server

# Build binary
go build -o server ./cmd/server
```

No test suite, linter config, or Makefile exists yet. The module is `web-backend`.

## Architecture

This is a Go admin-panel API backend using **Gin** + **sqlx** + **MySQL** with JWT auth.

**Directory layout:**

```
cmd/
  server/     - Entry point, route definitions
  migrate/    - Standalone DB migration tool (creates schema + default admin/admin123)
internal/
  config/     - Env var loading via godotenv (Config struct)
  database/   - MySQL connection pool via sqlx (global var database.DB)
  middleware/  - auth.go (JWT Bearer + RequireSuperAdmin), cors.go, logger.go
  models/     - Struct definitions + active-record-style data functions (parameterized on *sqlx.DB)
  handlers/   - HTTP handlers, one file per domain, each returns gin.HandlerFunc
  dto/        - Request/Response structs and standardized JSON response helpers
  services/   - Business logic: JWT token generation/parsing, Excel file parsing
docs/swagger/ - Empty, swagger annotations exist in handlers but aren't generated
```

**Key patterns:**

- All routes except `/api/health` and `/api/auth/login` require JWT Bearer auth (middleware `Auth`). Super-admin-only routes additionally use `RequireSuperAdmin`.
- The app enforces single-session login: login sets `users.is_active = 1`, and login is rejected if already active. On startup, all sessions are reset.
- Handlers are closure factories — e.g., `Login(cfg)` returns `gin.HandlerFunc`, which captures dependencies via closure.
- Models use `*sqlx.DB` as the first parameter to all data functions. No ORM, just raw SQL with sqlx.
- Excel import supports three target tables: `shops`, `foods`, `affairs`. The `/api/excel/preview` endpoint parses without inserting.
- Classtable (`/api/classtable`) is a stub — it logs the request and returns a placeholder. It requires integration with an external API.

**API response format:** All responses use `dto.Response` with `success`, `data`, `message`, and `total` fields. Use `dto.Success`, `dto.Error`, `dto.BadRequest`, etc. helpers.

**Config:** Loaded from `.env` (or env vars). Key variables: `PORT`, `GIN_MODE`, `DB_*`, `JWT_SECRET`, `JWT_EXPIRE_HOURS`, `FRONTEND_URL` (for CORS).
