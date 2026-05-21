-- +goose Up
CREATE TABLE games (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    enabled BOOLEAN DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE scores (
    id BIGSERIAL PRIMARY KEY,

    score BIGINT NOT NULL,
    player_name TEXT NOT NULL CHECK (char_length(player_name) <= 50),
    game_id BIGINT NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_scores_game_id_score_desc ON scores (game_id, score DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_scores_game_id_score_desc;
DROP TABLE scores;
DROP TABLE games;
