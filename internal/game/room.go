package game

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
	"sync"
	"time"
)

type Client interface{ Send(Event) bool }

type Room struct {
	Code       string
	duration   time.Duration
	tickRate   int
	mu         sync.RWMutex
	players    map[string]*Player
	clients    map[string]Client
	inputs     map[string]Input
	crystals   []Crystal
	seed       int64
	rng        *rand.Rand
	status     string
	startedAt  time.Time
	tick       uint64
	onComplete func(MatchResult)
	cancel     context.CancelFunc
}

func NewRoom(code string, duration time.Duration, tickRate int, onComplete func(MatchResult)) *Room {
	seed := seedFromCode(code)
	return &Room{
		Code:       code,
		duration:   duration,
		tickRate:   tickRate,
		players:    map[string]*Player{},
		clients:    map[string]Client{},
		inputs:     map[string]Input{},
		seed:       seed,
		rng:        rand.New(rand.NewSource(seed)),
		status:     "waiting",
		onComplete: onComplete,
	}
}

func (r *Room) AddPlayer(id, name string, client Client) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.players) >= 4 || r.status != "waiting" {
		return false
	}
	r.players[id] = &Player{ID: id, Name: name, X: 10 + float64(len(r.players))*15, Y: 50}
	r.clients[id] = client
	joined := *r.players[id]
	r.broadcastLocked(Event{Type: "player_joined", Data: joined})
	r.broadcastSnapshotLocked()
	return true
}

func (r *Room) RemovePlayer(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.clients, id)
	if r.status == "waiting" {
		delete(r.players, id)
		delete(r.inputs, id)
	}
	r.broadcastLocked(Event{Type: "player_left", Data: map[string]string{"playerId": id}})
	delete(r.inputs, id)
	r.broadcastSnapshotLocked()
}

func (r *Room) SetReady(id string, ready bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if player := r.players[id]; player != nil {
		player.Ready = ready
	}
	r.broadcastSnapshotLocked()
}

func (r *Room) ApplyInput(id string, input Input) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.status != "playing" || r.players[id] == nil {
		return
	}
	previous := r.inputs[id]
	if input.Sequence > previous.Sequence {
		r.inputs[id] = input
	}
}

func (r *Room) CanStart() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.status != "waiting" || len(r.players) < 2 {
		return false
	}
	for _, player := range r.players {
		if !player.Ready {
			return false
		}
	}
	return true
}

func (r *Room) Start() bool {
	r.mu.Lock()
	if r.status != "waiting" || len(r.players) < 2 {
		r.mu.Unlock()
		return false
	}
	for _, player := range r.players {
		if !player.Ready {
			r.mu.Unlock()
			return false
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel, r.status, r.startedAt = cancel, "playing", time.Now().UTC()
	r.spawnCrystalsLocked()
	r.mu.Unlock()
	go r.loop(ctx)
	return true
}

func (r *Room) loop(ctx context.Context) {
	ticker := time.NewTicker(time.Second / time.Duration(r.tickRate))
	defer ticker.Stop()
	deadline := time.NewTimer(r.duration)
	defer deadline.Stop()
	last := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case <-deadline.C:
			r.finish()
			return
		case now := <-ticker.C:
			dt := now.Sub(last).Seconds()
			last = now
			r.step(dt)
			r.broadcastSnapshot()
		}
	}
}

func (r *Room) step(dt float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.status != "playing" { return }
	r.tick++
	for id, input := range r.inputs {
		player := r.players[id]
		if player == nil {
			continue
		}
		length := math.Hypot(input.DX, input.DY)
		if length > 1 {
			input.DX /= length
			input.DY /= length
		}
		player.X = clamp(player.X+input.DX*MoveSpeed*dt, 0, ArenaWidth)
		player.Y = clamp(player.Y+input.DY*MoveSpeed*dt, 0, ArenaHeight)
	}
	r.collectCrystalsLocked()
}

func (r *Room) broadcastSnapshot() {
	r.mu.RLock()
	defer r.mu.RUnlock()
	r.broadcastSnapshotLocked()
}

// Caller holds r.mu for reading or writing.
func (r *Room) broadcastSnapshotLocked() {
	remaining := max(r.duration-time.Since(r.startedAt), 0)
	if r.status == "waiting" {
		remaining = r.duration
	}
	r.broadcastLocked(Event{Type: "snapshot", Data: Snapshot{
		Type:        "snapshot",
		RoomCode:    r.Code,
		Status:      r.status,
		Players:     clonePlayers(r.players),
		Crystals:    cloneCrystals(r.crystals),
		Seed:        r.seed,
		RemainingMS: remaining.Milliseconds(),
		Tick:        r.tick,
	}})
}

func (r *Room) finish() {
	r.mu.Lock()
	if r.status == "finished" {
		r.mu.Unlock()
		return
	}
	r.status = "finished"
	r.inputs = map[string]Input{}
	scores := make(map[string]int, len(r.players))
	for id, player := range r.players {
		scores[id] = player.Score
	}
	result := MatchResult{MatchID: newID(), Scores: scores, Started: r.startedAt}
	r.broadcastLocked(Event{Type: "match_finished", Data: result})
	r.mu.Unlock()
	if r.onComplete != nil {
		r.onComplete(result)
	}
}

func (r *Room) spawnCrystalsLocked() {
	r.crystals = make([]Crystal, CrystalCount)
	for index := range r.crystals {
		r.crystals[index] = r.newCrystal(index)
	}
}

func (r *Room) newCrystal(index int) Crystal {
	margin := PlayerRadius + CrystalRadius
	return Crystal{
		ID:    fmt.Sprintf("crystal-%d", index),
		X:     margin + r.rng.Float64()*(ArenaWidth-2*margin),
		Y:     margin + r.rng.Float64()*(ArenaHeight-2*margin),
		Value: CrystalScore,
	}
}

func (r *Room) collectCrystalsLocked() {
	collectionDistance := PlayerRadius + CrystalRadius
	for index, crystal := range r.crystals {
		collectorID := ""
		closest := math.Inf(1)
		for id, player := range r.players {
			distance := math.Hypot(player.X-crystal.X, player.Y-crystal.Y)
			if distance <= collectionDistance && (distance < closest || (distance == closest && (collectorID == "" || id < collectorID))) {
				collectorID, closest = id, distance
			}
		}
		if collectorID != "" {
			r.players[collectorID].Score += crystal.Value
			r.crystals[index] = r.newCrystal(index)
		}
	}
}

func (r *Room) broadcastLocked(event Event) {
	for _, client := range r.clients {
		client.Send(event)
	}
}

func clamp(v, low, high float64) float64 {
	return math.Max(low, math.Min(high, v))
}

func seedFromCode(code string) int64 {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(code))
	return int64(hash.Sum64() & ((1 << 63) - 1))
}

func clonePlayers(source map[string]*Player) map[string]*Player {
	result := make(map[string]*Player, len(source))
	for id, player := range source {
		copy := *player
		result[id] = &copy
	}
	return result
}

func cloneCrystals(source []Crystal) []Crystal {
	return append([]Crystal(nil), source...)
}
