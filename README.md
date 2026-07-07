# Game Jam Leaderboard

A small, self-hosted leaderboard backend for game jam games. Drop it behind your game, create a game entry, and let players submit and fetch scores.

It lets you:

- create and enable/disable games by name
- submit scores for a game
- fetch a game's leaderboard (sorted, paginated)
- rate limit score submissions and leaderboard reads per IP

## Tech stack

- Go 1.25
- [chi](https://github.com/go-chi/chi) router
- [sqlx](https://github.com/jmoiron/sqlx)
- PostgreSQL 16
- [goose](https://github.com/pressly/goose) for migrations
- Docker / Docker Compose
- Adminer (DB admin UI)
- [godotenv](https://github.com/joho/godotenv)

## Project layout

```
.
├── backend/            # Go HTTP API (main.go, ratelimitter.go)
├── migrations/          # goose SQL migrations
├── docker/
│   ├── backend.Dockerfile      # builds and runs the API
│   └── migrations.Dockerfile   # runs `goose up` against the DB
├── docker-compose-example.yml  # template — copy to docker-compose.yml
└── docker-compose.yml   # your local Postgres + Adminer + migrations + backend (gitignored)
```

There's no frontend in this repo — the backend is meant to be called from your game/site directly.

## Requirements

- Go 1.25+
- Docker + Docker Compose (recommended), or a standalone PostgreSQL 16+ instance

## Setup

### Option A: everything in Docker Compose (simplest)

Postgres, Adminer, the migrations job, and the backend itself all run as compose services.

```bash
cp docker-compose-example.yml docker-compose.yml
```

Edit `docker-compose.yml`:

- Set `POSTGRES_PASSWORD` under the `db` service.
- Set `GOOSE_DBSTRING` under `migrations` and `DATABASE_URL` under `backend` to match — use `db:5432` as the host (that's the internal compose network hostname/port), e.g. `postgresql://postgres:<password>@db:5432/postgres?sslmode=disable`.
- Set `ADMIN_SECRET` under `backend` to whatever secret you want to use for admin requests.

Then bring everything up:

```bash
docker compose up --build
```

This starts Postgres and Adminer, runs the migrations container once to apply the schema, and then starts the backend (it waits for migrations to finish successfully before starting).

> Running just `docker compose up --build migrations` will start Postgres and run the migration, but **not** Adminer or the backend, since neither is a dependency of the `migrations` service — use the plain `docker compose up --build` above to get everything.

- The API listens on [http://localhost:3000](http://localhost:3000).
- Adminer is at [http://localhost:8080](http://localhost:8080) (system: PostgreSQL, server: `db`, user `postgres`, password from `docker-compose.yml`).

Rebuild after changing backend code with `docker compose up --build backend`.

### Option B: run the backend locally (faster iteration)

Useful while actively developing, since `go run .` picks up changes instantly without a Docker rebuild.

Start just Postgres, Adminer, and migrations, and expose Postgres to your host so the locally-running backend can reach it:

```bash
cp docker-compose-example.yml docker-compose.yml
```

Add a `ports` entry under the `db` service (the example doesn't expose one by default):

```yaml
db:
  ...
  ports:
    - "5432:5432"
```

Fill in `POSTGRES_PASSWORD`/`GOOSE_DBSTRING` as above, then start everything except the backend:

```bash
docker compose up --build db adminer migrations
```

The backend loads a `.env` file automatically (via `godotenv`), so you don't need to export variables by hand:

```bash
cd backend
cp .env.example .env
```

Edit `backend/.env`, pointing at the host-mapped Postgres port this time:

```bash
DATABASE_URL="postgresql://postgres:<password>@localhost:5432/postgres?sslmode=disable"
ADMIN_SECRET="<pick-a-secret>"
```

Then run it:

```bash
go run .
```

The server listens on `http://localhost:3000`.

If you're not using Docker Compose at all, just point `DATABASE_URL` at any Postgres 16+ instance that already has the migrations applied (`goose up` from the `migrations/` directory).

## Environment variables

Read by the backend at startup (`backend/.env`, or the `backend` service's `environment:` block in Docker):

| Variable | Required | Purpose |
|---|---|---|
| `DATABASE_URL` | yes | PostgreSQL connection string |
| `ADMIN_SECRET` | no* | API key needed to create/update games. Without it, the create/update game endpoint is permanently locked out (not just unauthenticated) |

Read by the migrations container (set in `docker-compose.yml`):

| Variable | Purpose |
|---|---|
| `GOOSE_DRIVER` | Always `postgres` |
| `GOOSE_DBSTRING` | PostgreSQL connection string, reachable from inside the compose network |

## API

### Create or update a game

`PUT /games/`

Header:

- `X-API-Key: <ADMIN_SECRET>`

Body:

```json
{
  "game_name": "my-game",
  "enabled": true
}
```

Creates the game if it doesn't exist, or updates its `enabled` flag if it does (upsert on `game_name`).

Responses: `401 Unauthorized` if the API key is missing/wrong (or `ADMIN_SECRET` isn't set at all), `400 Bad Request` on malformed JSON.

### Get scores

`GET /games/{gameName}/scores?limit=20&offset=0`

Returns scores for the game, highest score first.

```json
[
  {
    "id": 1,
    "score": 1234,
    "player_name": "PlayerName",
    "game_id": 1,
    "created_at": "2026-07-07T12:00:00Z"
  }
]
```

Responses: `404 Not Found` if the game name doesn't exist, `403 Forbidden` if the game exists but is disabled, `429 Too Many Requests` if rate limited.

### Submit a score

`POST /games/{gameName}/scores`

Body:

```json
{
  "score": 1234,
  "player_name": "PlayerName"
}
```

Responses: `404 Not Found` / `403 Forbidden` (same rules as above), `400 Bad Request` on malformed JSON, `429 Too Many Requests` if rate limited.

## Rate limiting

Limits are tracked in memory, per client IP (not persisted, resets on backend restart):

- **Reads** (`GET /scores`): up to 50 requests per 30-second window. Exceeding it bans the IP from reads for 1 minute.
- **Writes** (`POST /scores`): at most 1 submission per 30 seconds per IP; extra requests get a `429` telling you how long to wait.

## Notes

- Player names are limited to 50 characters at the database level (`CHECK` constraint).
- Scores are stored as `BIGINT`; the API decodes them as 32-bit ints (`int32`).

## Future plans / potential improvements

- Add a profanity filter for player names if needed (may be better handled client-side before submission).
