package game

import "time"

const (
	ArenaWidth    = 100.0
	ArenaHeight   = 100.0
	MoveSpeed     = 18.0
	PlayerRadius  = 1.5
	CrystalRadius = 1.0
	CrystalScore  = 10
	CrystalCount  = 8
)

type Player struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Score int     `json:"score"`
	Ready bool    `json:"ready"`
}

type Input struct {
	DX       float64 `json:"dx"`
	DY       float64 `json:"dy"`
	Sequence uint64  `json:"sequence"`
}

type Crystal struct {
	ID    string  `json:"id"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Value int     `json:"value"`
}

type Snapshot struct {
	Type        string             `json:"type"`
	RoomCode    string             `json:"roomCode"`
	Status      string             `json:"status"`
	Players     map[string]*Player `json:"players"`
	Crystals    []Crystal          `json:"crystals"`
	Seed        int64              `json:"seed"`
	RemainingMS int64              `json:"remainingMs"`
	Tick        uint64             `json:"tick"`
}

type Event struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

type MatchResult struct {
	MatchID string         `json:"matchId"`
	Scores  map[string]int `json:"scores"`
	Started time.Time      `json:"startedAt"`
}
