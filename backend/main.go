package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"

	_ "github.com/lib/pq"
)

type Game struct {
	ID        int32     `db:"id"`
	Name      string    `db:"name"`
	Enabled   bool      `db:"enabled"`
	CreatedAt time.Time `db:"created_at"`
}

type Score struct {
	ID         int32     `db:"id"`
	Score      int32     `db:"score"`
	PlayerName string    `db:"player_name"`
	GameID     int32     `db:"game_id"`
	CreatedAt  time.Time `db:"created_at"`
}

type ScoreSubmit struct {
	Score      int32  `json:"score"`
	PlayerName string `json:"player_name"`
}

type GamePut struct {
	GameName string `json:"game_name"`
	Enabled  bool   `json:"enabled"`
}

const gameCtxKey string = "game"

var DB *sqlx.DB
var AdminSecret string

func main() {
	godotenv.Load()

	dburl, ok := os.LookupEnv("DATABASE_URL")
	if !ok {
		log.Fatal("DATABASE_URL environment variable not set.\n")
	}

	AdminSecret, ok = os.LookupEnv("ADMIN_SECRET")
	if !ok {
		log.Printf("[WARNING]: ADMIN_SECRET environment variable not set. You will not be able to add new games to the db without setting this.\n")
	}

	var err error
	DB, err = sqlx.Connect("postgres", dburl)
	if err != nil {
		log.Fatalf("Could not connect to database: %v \n", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Route("/games", func(r chi.Router) {
		r.Route("/{gameName}", func(r chi.Router) {
			r.Use(GameCtx)

			r.With(RateLimitGet).Get("/scores", getGameScores)
			r.With(RateLimitPost).Post("/scores", postGameScores)
		})

		r.With(AdminOnly).Put("/", putGames)
	})

	http.ListenAndServe(":3000", r)
}

func GameCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		gameName := chi.URLParam(r, "gameName")
		game := &Game{}
		err := DB.GetContext(ctx, game, "SELECT * FROM games WHERE name = $1", gameName)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		gameCtx := context.WithValue(ctx, gameCtxKey, game)
		next.ServeHTTP(w, r.WithContext(gameCtx))
	})
}

// Admin only middleware
func AdminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if AdminSecret == "" || apiKey != AdminSecret {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func getGameScores(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	game, ok := ctx.Value(gameCtxKey).(*Game)
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnprocessableEntity), http.StatusUnprocessableEntity)
		return
	}

	if !game.Enabled {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	limit := 20
	offset := 0

	// limit
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil {
			limit = v
		}
	}

	// offset
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil {
			offset = v
		}
	}

	scores := []Score{}
	err := DB.SelectContext(ctx, &scores, `
		SELECT * FROM scores WHERE game_id = $1
		ORDER BY score DESC
		LIMIT $2 OFFSET $3
	`, game.ID, limit, offset)

	if err != nil {
		fmt.Println(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(scores)
}

func postGameScores(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	game, ok := ctx.Value(gameCtxKey).(*Game)
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnprocessableEntity), http.StatusUnprocessableEntity)
		return
	}

	if !game.Enabled {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	var submit ScoreSubmit
	err := json.NewDecoder(r.Body).Decode(&submit)
	if err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	_, err = DB.ExecContext(ctx, "INSERT INTO scores (score, player_name, game_id) VALUES ($1, $2, $3)", submit.Score, submit.PlayerName, game.ID)
	if err != nil {
		log.Printf("failed to insert score to game %s error=%v\n", game.Name, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func putGames(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var submit GamePut
	err := json.NewDecoder(r.Body).Decode(&submit)
	if err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	_, err = DB.ExecContext(ctx, `
		INSERT INTO games (name, enabled)
		VALUES ($1, $2)
		ON CONFLICT (name)
		DO UPDATE SET enabled = EXCLUDED.enabled
	`, submit.GameName, submit.Enabled)

	if err != nil {
		log.Printf("Failed to insert/update game name: %s error=%v\n", submit.GameName, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}
