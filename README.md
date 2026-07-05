# Go Initiative Tracker

A web-based initiative tracker for tabletop role-playing games. A Go (`net/http`)
backend exposes a JSON API over PostgreSQL; a React 19 + Material UI frontend
(built with Vite) consumes it.

## Features

- **Characters** — create a reusable library of player characters (name, AC,
  to-hit modifier, max HP).
- **Encounters** — group characters into encounters; each encounter tracks its
  own initiative, current HP, and active turn per character.
- **NPC templates** — define reusable monsters/NPCs and drop them into an
  encounter on demand.
- **Combat tracking** — sort by initiative, advance turns (press <kbd>Space</kbd>),
  and apply quick attack/heal actions with live HP previews
  (<kbd>Enter</kbd> = attack, <kbd>Shift</kbd>+<kbd>Enter</kbd> = heal).
- **Combat log** — a per-encounter ledger of attacks, heals, and notes.
- **Discord login** — optional OAuth2 sign-in scopes characters and encounters
  to the logged-in user.

## Repository layout

```
go-initiative-tracker/
  backend/      # Go API server (module: go-initiative-tracker)
    main.go     #   HTTP handlers and bootstrap
    dao/        #   database access objects (one per table/concern)
  frontend/     # React + Vite single-page app
  start.sql     # PostgreSQL schema + seed data
  docker-compose.dev.yml   # hot-reloading dev stack (air + vite)
  docker-compose.yml       # production-style build
  .env.example  # documented environment variables
```

## Quick start (Docker, recommended)

The dev compose file runs Postgres (seeded from `start.sql`), the Go backend with
hot reload (`air`), and the Vite dev server.

```bash
cp .env.example .env   # fill in DISCORD_* and DB_PASSWORD
docker compose -f docker-compose.dev.yml up --build
```

- Frontend: <http://localhost:5173>
- Backend API: <http://localhost:8080>

## Manual setup

### Prerequisites

- Go 1.23+
- Node.js 20+ (22 recommended)
- PostgreSQL 16+

### 1. Database

Create a database and load the schema + seed data:

```bash
createdb initiative_db
psql -d initiative_db -f start.sql
```

### 2. Backend

```bash
cd backend
cp ../.env.example .env   # adjust DB_* values for your local Postgres
go mod download
go run .
```

The server reads configuration from environment variables (see
[`.env.example`](.env.example)); a local `.env` file is loaded automatically.
It listens on `PORT` (default `8080`).

### 3. Frontend

```bash
cd frontend
npm install
npm run dev
```

Vite proxies requests beginning with `/api` to the backend (see
[`frontend/vite.config.ts`](frontend/vite.config.ts)), so no extra configuration
is needed in development.

## Configuration

All configuration is via environment variables, documented in
[`.env.example`](.env.example). Highlights:

| Variable | Purpose | Default |
| --- | --- | --- |
| `DB_HOST`/`DB_PORT`/`DB_USER`/`DB_PASSWORD`/`DB_NAME` | Postgres connection | — |
| `SSLMODE` | Postgres SSL mode | `disable` |
| `PORT` | Backend listen port | `8080` |
| `FRONTEND_URL` | Frontend origin (redirects + CORS) | `http://localhost:5173` |
| `SECURE_COOKIES` | Mark auth cookies `Secure` (use in production) | `false` |
| `DISCORD_CLIENT_ID`/`DISCORD_CLIENT_SECRET`/`DISCORD_REDIRECT_URL` | Discord OAuth | — |

## Testing & linting

**Backend** (run from `backend/`):

```bash
go test ./...     # unit tests (uses go-sqlmock, no live DB required)
go vet ./...
gofmt -l .        # should print nothing
```

**Frontend** (run from `frontend/`):

```bash
npm run lint      # eslint
npx tsc -b        # type-check
npm run build     # production build
```

CI ([`.github/workflows/ci.yml`](.github/workflows/ci.yml)) runs all of the
above on every push and pull request to `main`.

## License

MIT
