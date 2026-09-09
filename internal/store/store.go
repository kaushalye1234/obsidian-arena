package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type LeaderboardEntry struct {
	PlayerID   string `json:"playerId"`
	DisplayName string `json:"displayName"`
	BestScore  int    `json:"bestScore"`
	TotalScore int64  `json:"totalScore"`
	Wins       int    `json:"wins"`
}

type Store struct{ pool *pgxpool.Pool }

func New(ctx context.Context, url string) (*Store, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil { return nil, err }
	if err = pool.Ping(ctx); err != nil { pool.Close(); return nil, err }
	return &Store{pool: pool}, nil
}

func (s *Store) Close() { s.pool.Close() }

func (s *Store) EnsurePlayer(ctx context.Context, name string) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO players (display_name) VALUES ($1)
		ON CONFLICT (display_name) DO UPDATE SET updated_at = NOW()
		RETURNING id`, name).Scan(&id)
	return id, err
}

func (s *Store) Leaderboard(ctx context.Context, limit int) ([]LeaderboardEntry, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, display_name, best_score, total_score, wins
		FROM players ORDER BY best_score DESC, total_score DESC LIMIT $1`, limit)
	if err != nil { return nil, err }
	defer rows.Close()
	entries := make([]LeaderboardEntry, 0, limit)
	for rows.Next() {
		var entry LeaderboardEntry
		if err := rows.Scan(&entry.PlayerID, &entry.DisplayName, &entry.BestScore, &entry.TotalScore, &entry.Wins); err != nil { return nil, err }
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (s *Store) Progress(ctx context.Context, playerID string) (json.RawMessage, error) {
	var progress []byte
	err := s.pool.QueryRow(ctx, `SELECT progress FROM players WHERE id = $1`, playerID).Scan(&progress)
	return json.RawMessage(progress), err
}

func (s *Store) SaveProgress(ctx context.Context, playerID string, progress json.RawMessage) error {
	_, err := s.pool.Exec(ctx, `UPDATE players SET progress = $2, updated_at = NOW() WHERE id = $1`, playerID, progress)
	return err
}

func (s *Store) SaveMatch(ctx context.Context, matchID, roomCode string, started time.Time, scores map[string]int) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil { return err }
	defer tx.Rollback(ctx)
	var winnerID string
	maxScore := -1
	for playerID, score := range scores { if score > maxScore { winnerID, maxScore = playerID, score } }
	if _, err = tx.Exec(ctx, `INSERT INTO matches (id, room_code, winner_id, started_at) VALUES ($1,$2,$3,$4)`, matchID, roomCode, winnerID, started); err != nil { return err }
	for playerID, score := range scores {
		if _, err = tx.Exec(ctx, `INSERT INTO match_scores (match_id, player_id, score) VALUES ($1,$2,$3)`, matchID, playerID, score); err != nil { return err }
		won := playerID == winnerID
		if _, err = tx.Exec(ctx, `UPDATE players SET total_score=total_score+$2, best_score=GREATEST(best_score,$2), games_played=games_played+1, wins=wins+$3, updated_at=NOW() WHERE id=$1`, playerID, score, boolInt(won)); err != nil { return err }
	}
	return tx.Commit(ctx)
}

func boolInt(value bool) int { if value { return 1 }; return 0 }
