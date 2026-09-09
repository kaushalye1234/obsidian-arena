CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS players (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    display_name VARCHAR(24) NOT NULL UNIQUE,
    total_score BIGINT NOT NULL DEFAULT 0,
    best_score INTEGER NOT NULL DEFAULT 0,
    games_played INTEGER NOT NULL DEFAULT 0,
    wins INTEGER NOT NULL DEFAULT 0,
    progress JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS matches (
    id UUID PRIMARY KEY,
    room_code VARCHAR(8) NOT NULL,
    winner_id UUID REFERENCES players(id),
    started_at TIMESTAMPTZ NOT NULL,
    ended_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS match_scores (
    match_id UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    player_id UUID NOT NULL REFERENCES players(id),
    score INTEGER NOT NULL CHECK (score >= 0),
    PRIMARY KEY (match_id, player_id)
);

CREATE INDEX IF NOT EXISTS idx_players_best_score ON players(best_score DESC);
CREATE INDEX IF NOT EXISTS idx_players_total_score ON players(total_score DESC);
