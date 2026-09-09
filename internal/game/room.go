package game

import (
	"context"
	"math"
	"sync"
	"time"
)

type Client interface { Send(Event) bool }

type Room struct {
	Code       string
	duration   time.Duration
	tickRate   int
	mu         sync.RWMutex
	players    map[string]*Player
	clients    map[string]Client
	inputs     map[string]Input
	status     string
	startedAt  time.Time
	tick       uint64
	onComplete func(MatchResult)
	cancel     context.CancelFunc
}

func NewRoom(code string, duration time.Duration, tickRate int, onComplete func(MatchResult)) *Room {
	return &Room{Code: code, duration: duration, tickRate: tickRate, players: map[string]*Player{}, clients: map[string]Client{}, inputs: map[string]Input{}, status: "waiting", onComplete: onComplete}
}

func (r *Room) AddPlayer(id, name string, client Client) bool {
	r.mu.Lock(); defer r.mu.Unlock()
	if len(r.players) >= 4 || r.status != "waiting" { return false }
	r.players[id] = &Player{ID: id, Name: name, X: 10 + float64(len(r.players))*15, Y: 50}
	r.clients[id] = client
	joined := *r.players[id]
	r.broadcastLocked(Event{Type: "player_joined", Data: joined})
	return true
}

func (r *Room) RemovePlayer(id string) {
	r.mu.Lock(); defer r.mu.Unlock()
	delete(r.clients, id)
	if r.status == "waiting" { delete(r.players, id); delete(r.inputs, id) }
	r.broadcastLocked(Event{Type: "player_left", Data: map[string]string{"playerId": id}})
}

func (r *Room) SetReady(id string, ready bool) {
	r.mu.Lock(); defer r.mu.Unlock()
	if player := r.players[id]; player != nil { player.Ready = ready }
}

func (r *Room) ApplyInput(id string, input Input) {
	r.mu.Lock(); defer r.mu.Unlock()
	if r.status != "playing" || r.players[id] == nil { return }
	previous := r.inputs[id]
	if input.Sequence > previous.Sequence { r.inputs[id] = input }
}

func (r *Room) CanStart() bool {
	r.mu.RLock(); defer r.mu.RUnlock()
	if r.status != "waiting" || len(r.players) < 2 { return false }
	for _, player := range r.players { if !player.Ready { return false } }
	return true
}

func (r *Room) Start() bool {
	r.mu.Lock()
	if r.status != "waiting" || len(r.players) < 2 { r.mu.Unlock(); return false }
	for _, player := range r.players { if !player.Ready { r.mu.Unlock(); return false } }
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel, r.status, r.startedAt = cancel, "playing", time.Now().UTC()
	r.mu.Unlock()
	go r.loop(ctx)
	return true
}

func (r *Room) loop(ctx context.Context) {
	ticker := time.NewTicker(time.Second / time.Duration(r.tickRate)); defer ticker.Stop()
	deadline := time.NewTimer(r.duration); defer deadline.Stop()
	last := time.Now()
	for {
		select {
		case <-ctx.Done(): return
		case <-deadline.C: r.finish(); return
		case now := <-ticker.C:
			dt := now.Sub(last).Seconds(); last = now
			r.step(dt); r.broadcastSnapshot()
		}
	}
}

func (r *Room) step(dt float64) {
	r.mu.Lock(); defer r.mu.Unlock(); r.tick++
	for id, input := range r.inputs {
		player := r.players[id]; if player == nil { continue }
		length := math.Hypot(input.DX, input.DY)
		if length > 1 { input.DX /= length; input.DY /= length }
		player.X = clamp(player.X+input.DX*MoveSpeed*dt, 0, ArenaWidth)
		player.Y = clamp(player.Y+input.DY*MoveSpeed*dt, 0, ArenaHeight)
		// MVP scoring: one point per server tick while moving.
		if math.Abs(input.DX)+math.Abs(input.DY) > 0 { player.Score++ }
	}
}

func (r *Room) broadcastSnapshot() {
	r.mu.RLock(); defer r.mu.RUnlock()
	remaining := max(r.duration-time.Since(r.startedAt), 0)
	r.broadcastLocked(Event{Type: "snapshot", Data: Snapshot{Type: "snapshot", RoomCode: r.Code, Status: r.status, Players: clonePlayers(r.players), RemainingMS: remaining.Milliseconds(), Tick: r.tick}})
}

func (r *Room) finish() {
	r.mu.Lock(); r.status = "finished"
	scores := make(map[string]int, len(r.players))
	for id, player := range r.players { scores[id] = player.Score }
	result := MatchResult{MatchID: newID(), Scores: scores, Started: r.startedAt}
	r.broadcastLocked(Event{Type: "match_finished", Data: result}); r.mu.Unlock()
	if r.onComplete != nil { r.onComplete(result) }
}

func (r *Room) broadcastLocked(event Event) { for _, client := range r.clients { client.Send(event) } }
func clamp(v, low, high float64) float64 { return math.Max(low, math.Min(high, v)) }
func clonePlayers(source map[string]*Player) map[string]*Player {
	result := make(map[string]*Player, len(source))
	for id, player := range source { copy := *player; result[id] = &copy }
	return result
}
