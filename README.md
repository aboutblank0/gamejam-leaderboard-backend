# Game Jam Leaderboard Backend

Simple backend for game jam leaderboards.

It lets you:

- create and enable/disable games by name
- submit scores for a game
- fetch a game's leaderboard
- rate limit score submissions and leaderboard reads

## Requirements

- Go 1.25+
- PostgreSQL 16+

## Environment variables

The backend reads these at startup:

- `DATABASE_URL` — required. PostgreSQL connection string.
- `ADMIN_SECRET` — optional, but required if you want to create or update games.

## Run locally

### Option 1: Docker + Docker Compose

The repo includes `docker-compose-example.yml` for PostgreSQL, Adminer, and migrations.

1. Fill in `POSTGRES_PASSWORD` and `GOOSE_DBSTRING` in `docker-compose-example.yml`.
2. Start the database and Adminer:

   ```bash
   docker compose -f docker-compose-example.yml up -d db adminer
   ```

3. Run the migrations:

   ```bash
   docker compose -f docker-compose-example.yml up --build migrations
   ```

4. In another shell, start the backend:

   ```bash
   cd backend
   DATABASE_URL="postgresql://postgres:<password>@localhost:5432/postgres?sslmode=disable" \
   ADMIN_SECRET="<secret>" \
   go run .
   ```

Adminer is available at `http://localhost:8080`.

### Option 2: Run the backend directly

1. Start PostgreSQL.
2. Apply the SQL migration in `migrations/20260519110512_add_games_scores_tables.sql`.
3. Set `DATABASE_URL` and, if needed, `ADMIN_SECRET`.
4. Run the backend from the `backend` directory:

   ```bash
   go run .
   ```

The server listens on `http://localhost:3000`.

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

### Get scores

`GET /games/{gameName}/scores?limit=20&offset=0`

Returns scores ordered by highest score first.

### Submit a score

`POST /games/{gameName}/scores`

Body:

```json
{
  "score": 1234,
  "player_name": "PlayerName"
}
```

## Notes

- Score submissions are rate limited per IP.
- Leaderboard reads are also rate limited per IP.
- Player names are limited to 50 characters in the database.

## Future plans / potential improvements

- Add a profanity filter for player names if needed.
- In practice, this may be better handled on the frontend before submission.
