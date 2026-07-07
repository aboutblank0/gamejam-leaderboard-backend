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
│   └── migrations.Dockerfile   # runs `goose up` against the DB
├── docker-compose-example.yml  # template — copy to docker-compose.yml
└── docker-compose.yml   # your local Postgres + Adminer + migrations (gitignored)
```

There's no frontend in this repo — the backend is meant to be called from your game/site directly.

## Requirements

- Go 1.25+
- Docker + Docker Compose (recommended), or a standalone PostgreSQL 16+ instance

## Setup

### 1. Start Postgres + Adminer + run migrations

```bash
cp docker-compose-example.yml docker-compose.yml
```

Edit `docker-compose.yml`:

- Set `POSTGRES_PASSWORD` under the `db` service.
- Set `GOOSE_DBSTRING` under the `migrations` service to match (host is `db` since it talks to Postgres over the compose network).
- The example file doesn't expose Postgres to your host machine. Since the backend itself runs outside Docker (there's no backend service in the compose file), add a `ports` entry under `db` so you can connect to it, e.g.:

  ```yaml
  db:
    ...
    ports:
      - "5432:5432"
  ```

Then bring everything up (this starts Postgres and Adminer, and runs the migrations container once to apply the schema):

```bash
docker compose up --build
```

> Running just `docker compose up --build migrations` will start Postgres and run the migration, but **not** Adminer, since Adminer isn't a dependency of the `migrations` service — use the plain `docker compose up --build` above to get all three.

Adminer is then available at [http://localhost:8080](http://localhost:8080) (system: PostgreSQL, server: `db` if browsing from inside the compose network, or `localhost:<port>` from your host; user `postgres`, password from `docker-compose.yml`).

### 2. Configure and run the backend

The backend loads a `.env` file automatically (via `godotenv`), so you don't need to export variables by hand.

```bash
cd backend
cp .env.example .env
```

Edit `backend/.env`:

```bash
DATABASE_URL="postgresql://postgres:<password>@localhost:<port>/postgres?sslmode=disable"
ADMIN_SECRET="<pick-a-secret>"
```

Then run it:

```bash
go run .
```

The server listens on `http://localhost:3000`.

If you're not using Docker Compose, just point `DATABASE_URL` at any Postgres 16+ instance that already has the migrations applied (`goose up` from the `migrations/` directory, or via `docker compose up --build migrations`).

## Environment variables

Read by the backend at startup (`backend/.env` or the real environment):

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
    "ID": 1,
    "Score": 1234,
    "PlayerName": "PlayerName",
    "GameID": 1,
    "CreatedAt": "2026-07-07T12:00:00Z"
  }
]
```

> Note the field names are capitalized — the response structs don't define `json` tags, so Go's default (exported field name) is used as-is.

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
- Add `json` tags to response structs for consistent snake_case output.
