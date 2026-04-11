# Copilot Instructions for Cinema Booking System

## Overview

A Go backend for a high-concurrency cinema ticket booking system that prevents double-bookings using PostgreSQL's `ON CONFLICT` constraints and partial unique indexes. The system uses a "hold then confirm" pattern with automatic hold expiration.

**Key Problem Solved**: When two users click "Book" on the same seat simultaneously, only one succeeds—the other gets an immediate `ErrSeatTaken` error.

## Architecture

### Project Structure

```
cmd/main.go              # HTTP server setup, route registration, graceful shutdown
internal/
├── adapters/postgres/   # Database connection pool, migrations, error handling
├── booking/             # Seat booking domain (hold/confirm/release workflow)
├── movie/               # Movie listing domain
├── config/              # Environment variable loading
└── utils/               # JSON response helper
migrations/              # SQL migrations (numbered, up/down files)
static/                  # Frontend assets
```

### Domain-Driven Design Pattern

Each domain (booking, movie) follows this structure:
- **domain.go**: Core types, interfaces, and error constants
- **service.go**: Business logic thin wrapper around store
- **handler.go**: HTTP handlers with JSON encoding/request parsing
- **postgres_store.go**: Store interface implementation (database queries)

### Request Flow

1. HTTP handler (handler.go) parses request → validates userID
2. Calls service method (service.go)
3. Service delegates to store (postgres_store.go)
4. Store executes database query with context
5. Handler returns JSON response or error

### Concurrency Control

**Problem**: Double-booking race condition in read-then-write pattern.

**Solution**: PostgreSQL `ON CONFLICT DO NOTHING` with partial unique index
- Table: `bookings` with columns: id, movie_id, seat_id, user_id, status (held/confirmed), expires_at
- Index: `idx_active_seat` is a UNIQUE partial index on `(movie_id, seat_id)` WHERE status IN ('held', 'confirmed')
- Query: `INSERT ... ON CONFLICT (movie_id, seat_id) WHERE status IN ('held', 'confirmed') DO NOTHING`
- Returns 0 rows if conflict exists → client gets `ErrSeatTaken` immediately

### Booking Lifecycle

1. **Hold** (POST /movies/{movieID}/seats/{seatID}/hold): Insert with status='held', ttl=2min (configurable)
2. **Confirm** (PUT /sessions/{sessionID}/confirm): UPDATE status from 'held' to 'confirmed', clear expires_at
3. **Release** (DELETE /sessions/{sessionID}): DELETE if status='held' (cancels hold)
4. **Expiry**: Background goroutine cleans expired 'held' bookings every 30 seconds

## Build, Test, and Run

### Build
```bash
go build -o cinema ./cmd/main.go
```

### Run (requires PostgreSQL)
```bash
# Start PostgreSQL
docker-compose up redis redis-commander

# Set environment variables (optional, uses defaults)
export DATABASE_URL="postgres://cinema:cinema@localhost:5432/cinema?sslmode=disable"
export SERVER_PORT="8080"
export HOLD_TTL="2m"

go run ./cmd/main.go
```

The server applies migrations automatically on startup and prints connection stats.

### Test
```bash
go test ./internal/...
```

No existing tests in repo; add tests to `{domain}_test.go` files (e.g., `internal/booking/postgres_store_test.go`).

## Key Conventions

### Error Handling
- Package-level error constants for domain errors (e.g., `var ErrSeatTaken = errors.New("seat already taken")`)
- Store methods check `pgx.ErrNoRows` explicitly for not-found conditions
- HTTP handlers map domain errors to appropriate status codes (409 Conflict for `ErrSeatTaken`)

### Configuration
- All env-var loading in `config.Load()` with sensible defaults
- Database URL and TTL values parameterized (not hardcoded)
- Server port configurable via `SERVER_PORT` env var

### Database Queries
- Always use parameterized queries (e.g., `$1, $2`) to prevent SQL injection
- Pass `context.Context` to all pool methods for cancellation support
- Use `Scan` for single-row results, `Query` + loop for multi-row

### JSON Serialization
- Use `encoding/json` struct tags for field name mapping (e.g., `json:"seatID"`)
- Utility function `utils.WriteJSON(w, status, data)` for consistent response format

### Background Workers
- Use `time.NewTicker` for periodic tasks (e.g., hold expiry)
- Always listen to `ctx.Done()` for graceful shutdown
- Log completion on context cancellation

## Database Setup

Migrations are automatically applied on server startup (`postgres.Migrate(ctx, pool, "migrations")`).

Migration files:
- `001_create_movies.up.sql`: Creates movies table with id, title, rows, seats_per_row
- `002_create_bookings.up.sql`: Creates bookings table + `idx_active_seat` partial index (the core of double-booking prevention)
- `003_seed_movies.up.sql`: Seeds sample movie data

Each migration has a corresponding `.down.sql` file for rollback.

## MCP Servers

### PostgreSQL MCP Server

To enable direct database queries from Copilot:

```json
{
  "postgresql": {
    "command": "npx",
    "args": ["@modelcontextprotocol/server-postgresql"],
    "env": {
      "DATABASE_URL": "postgres://cinema:cinema@localhost:5432/cinema?sslmode=disable"
    }
  }
}
```

This allows Copilot to:
- Query the bookings and movies tables
- Inspect the schema and indexes
- Test migrations and data queries
- Verify the partial unique index constraints work correctly

Start PostgreSQL first: `docker-compose up redis redis-commander` (or set up your own Postgres instance).

## Maintenance

**Keep this file updated** whenever architecture or patterns change:
- Adding new domain packages (follow the domain-driven structure: domain.go, service.go, handler.go, postgres_store.go)
- Modifying the booking lifecycle or concurrency strategy
- Adding new configuration options or environment variables
- Introducing new dependencies or external services
- Changing HTTP routes or API contracts
- Updating migration strategy or database schema constraints

Treat this file as the single source of truth for how the system works. Future Copilot sessions depend on it to understand the codebase correctly.
