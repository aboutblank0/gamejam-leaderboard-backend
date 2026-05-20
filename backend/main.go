package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
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
	id          int32     `db:"id"`
	score       int32     `db:"score"`
	player_name string    `db:"player_name"`
	game_id     int32     `db:"game_id"`
	created_at  time.Time `db:"created_at"`
}

func main() {
	godotenv.Load()

	dburl, ok := os.LookupEnv("DATABASE_URL")
	if !ok {
		log.Fatal("DATABASE_URL environment variable not set.\n")
	}

	db, err := sqlx.Connect("postgres", dburl)
	if err != nil {
		log.Fatalf("Could not connect to database: %v \n", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		games := []Game{}
		db.Select(&games, "SELECT * FROM games")
		w.Write([]byte(fmt.Sprintf("%v# \n", games)))
	})

	http.ListenAndServe(":3000", r)
}
