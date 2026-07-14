# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A web-based TTRPG initiative tracker. A Go (`net/http`) backend exposes a JSON API
over PostgreSQL; a React 19 + Material UI frontend (Vite) consumes it. Optional
Discord OAuth2 scopes characters/encounters to the signed-in user.

## Commands

Run backend commands from `backend/`, frontend commands from `frontend/`.

**Backend (Go 1.25):**
```bash
go build ./...            # build
go vet ./...              # vet
gofmt -l .                # list unformatted files; CI fails if any (gofmt is required, not optional)
go test ./...             # run all tests
go test -run TestName ./  # run a single test
go run .                  # run server (needs DB_* env vars; loads ./.env if present)
```

**Frontend (Node 22):**
```bash
npm run dev     # Vite dev server on :5173
npm run lint    # eslint
npx tsc -b      # type-check
npm run build   # tsc -b && vite build
```
> Node/npm/npx are only on PATH in the PowerShell tool here, not Bash.

**Full dev stack (recommended):** `docker compose -f docker-compose.dev.yml up --build`
brings up Postgres (seeded from `start.sql`), the backend with `air` hot reload, and
the Vite dev server. CI (`.github/workflows/ci.yml`) runs the backend and frontend
command sets above on every push/PR to `main`.

## Architecture

**Request path & the `/api` rewrite.** The backend registers routes at the *root*
(`/characters`, `/encounters`, `/npcs/templates`, …) — there is no `/api` prefix on
the server. The frontend calls everything under `/api/...`, and the Vite proxy
(`vite.config.ts`) strips `/api` before forwarding to the backend. So a frontend
`/api/characters` hits the backend's `/characters`. Keep this in mind when adding
routes: register at root in `backend/main.go`, call with `/api` from the frontend.

**Backend layering.** `backend/main.go` holds all HTTP handlers, OAuth flow, CORS
middleware (`loggingMiddleware`), and bootstrap. Data access is isolated in
`backend/dao/` — one file per table/concern, each exposing an **interface + impl +
`NewXxxDAO(db)` constructor**. Handlers talk only to DAO interfaces (package-level
vars wired in `initializeApp`). Combat ordering/turn logic lives in the DAO layer
(`encounter_character_dao.go`: `StartCombat`, `AdvanceTurn`, `ResetCombat` run inside
transactions, ordering by `initiative DESC, character_id ASC`).

**Server-side mutable global state (important gotcha).** `main.go` keeps
package-level globals — `selectedEncounterID`, `characters`, `encounters`. The
"selected encounter" is process-wide, **not** per-user or per-session. Several
handlers read/write these globals (e.g. `apiCharactersHandler` sets
`selectedEncounterID` from a query param, `saveCharacterHandler` appends to the
`characters` slice). Treat this as shared state when reasoning about concurrent
requests; prefer passing IDs explicitly (many endpoints accept `encounter_id` in the
body/query) over relying on the global.

**Auth model.** Discord OAuth2 (`/login/discord` → `/auth/discord/callback`) sets
HttpOnly cookies `discord_user`, `discord_id`, `discord_avatar` — there is no JWT or
session store. Ownership is enforced by matching the `discord_id` cookie against a
row's `owner_id` (see `*ByOwner` DAO methods and `getDiscordIDFromRequest`). When
logged out, list endpoints fall back to returning all rows. `secureCookies` is gated
by `SECURE_COOKIES=true`.

**Frontend.** `App.tsx` is a single-page view switcher (`characters | encounters |
combat | npcs`) with no router — state, not URLs. Components in `src/components/`,
data-fetching in `src/hooks/` (`useCharacters`, `useEncounters`, `useCombatLog`,
`useNpcTemplates`). All network calls go through `src/lib/http.ts`
(`apiGet`/`apiGetArray`/`apiPost`), which sends `credentials: "include"` (for the
Discord cookie) and enforces the `{ status: "success", message? }` envelope that
mutation endpoints return. Uses the React Compiler (babel plugin) — avoid manual
`useMemo`/`useCallback` micro-optimizations that fight it.

**Database.** Schema + seed data in `start.sql` (tables: `users`, `npc_templates`,
`characters`, `encounters`, `encounter_ledger`, `encounter_characters`,
`encounter_users`). Postgres containers auto-load it via
`/docker-entrypoint-initdb.d`; for manual setup, `psql -f start.sql`.

**Migrations.** `start.sql` is the baseline that seeds a *fresh* database only
(initdb runs it just once, on an empty data dir). Every schema change *after* the
baseline is an additive, versioned migration under `backend/migrations/*.sql`
(goose format: `NNNNN_name.sql` with `-- +goose Up`/`Down` sections). These are
embedded into the binary (`migrations.go`) and applied automatically at startup
by `runMigrations` after the DB is reachable, so fresh and long-lived databases
converge instead of drifting. **Do not hand-patch a running DB's schema** — add a
migration. Write them defensively (`ADD COLUMN IF NOT EXISTS`) so they no-op on
databases that already have the change. goose tracks applied versions in
`goose_db_version`.

## Testing

Backend tests use `httptest` for handlers and **go-sqlmock** for the DAO layer (no
real DB). `TestMain` in `main_test.go` installs a shared mock `db` and calls
`initializeApp`, expecting the two bootstrap `SELECT`s — when adding handlers/DAO
methods that change startup queries, update those `mock.ExpectQuery` expectations or
tests will fail. There is no frontend test runner; the frontend CI gate is
lint + `tsc -b` + build.

## Configuration

All config is environment variables (documented in `.env.example`; `backend/main.go`
loads a local `.env` automatically, env vars take precedence). Key ones: `DB_HOST`,
`DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `SSLMODE` (default `disable`), `PORT`
(default `8080`), `FRONTEND_URL`, `SECURE_COOKIES`, and `DISCORD_CLIENT_ID` /
`DISCORD_CLIENT_SECRET` / `DISCORD_REDIRECT_URL` for OAuth.
