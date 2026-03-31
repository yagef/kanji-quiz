# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Development (Docker recommended)
```bash
docker-compose -f docker-compose.dev.yml up
```
Starts the API on `:8080` with hot reload (Air) and PostgreSQL.

### Local development (without Docker)
```bash
# Generate Templ templates (required before build)
templ generate

# Build and run
go build -o ./tmp/app ./server && ./tmp/app

# Hot reload (requires Air installed)
air
```

### Environment setup
Copy `.env` and ensure these are set before running:
- `DATABASE_URL` — PostgreSQL connection string
- `SESSION_AUTH_KEY` — HMAC key for session cookies
- `SESSION_ENCRYPT_KEY` — AES key for session data
- `ADMIN_PASS` — admin login password
- `SERVER_BASE_URL` — base URL used in QR code generation

## Architecture

### Stack
- **Go + Gin** — HTTP server and routing
- **Templ** — server-side HTML templates (`.templ` files compiled to `*_templ.go`)
- **pgx/v5** — PostgreSQL driver (no ORM)
- **Gorilla WebSocket** — real-time communication
- **Gorilla Sessions** — cookie-based session management

### Layers

**`server/handlers/`** — HTTP request handlers (Gin context, session reads, call repositories, render templates). `auth.go` provides session middleware for both admin and user routes.

**`server/repository/`** — All DB queries using raw SQL via pgx. No query builder or ORM.

**`server/ws/`** — Real-time quiz engine:
- `manager.go` — registry of active `SessionHub`s, keyed by session ID
- `hub.go` — manages WebSocket clients for one quiz session; handles broadcast
- `engine.go` — quiz state machine; drives question progression, countdowns, and phase transitions
- `messages.go` — WebSocket message type definitions (JSON)

**`server/model/`** — Shared data structs. `session_state.go` defines quiz phases (`Waiting`, `Countdown`, `Answering`, `Paused`, `Finished`) and shuffled question/answer state stored in memory during a session.

**`pages/`** — Templ templates. `pages/admin/` for the admin UI, `pages/user/` for participant UI. Generated `*_templ.go` files are git-ignored and must be regenerated with `templ generate`.

**`db/init.sql`** — Full schema + seed data. Tables: `quizzes`, `quiz_sessions`, `questions`, `answers`, `answer_types`, `users`, `participants`, `submissions`.

### Real-time flow
1. Admin creates a session → generates QR code linking to `/user/join/:sessionID`
2. Participant scans QR, enters name → creates `participant` record → connects WebSocket at `/ws/participants/:participantID`
3. Admin starts quiz → `Engine` pushes questions to `SessionHub` → broadcast to all participant WebSocket clients
4. Participants answer → submissions recorded with correctness, time, and score
5. Admin advances rounds; engine manages phase transitions and timing

### Key patterns
- Templ templates must be regenerated whenever `.templ` files change (`templ generate`)
- Session state (shuffled questions, current round, phase) is held in-memory inside `Engine`, not in the DB
- The debugger (Delve) listens on port `40000` in the dev Docker setup
