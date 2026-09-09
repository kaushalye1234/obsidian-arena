package game

import (
	"testing"
	"time"
)

type fakeClient struct{ events []Event }
func (f *fakeClient) Send(e Event) bool { f.events = append(f.events, e); return true }

func TestRoomRequiresTwoReadyPlayers(t *testing.T) {
	r := NewRoom("ABC123", time.Second, 20, nil)
	r.AddPlayer("1", "Ada", &fakeClient{})
	if r.CanStart() { t.Fatal("one player must not start a match") }
	r.AddPlayer("2", "Linus", &fakeClient{})
	r.SetReady("1", true); r.SetReady("2", true)
	if !r.CanStart() { t.Fatal("two ready players should start") }
}

func TestMovementIsClampedToArena(t *testing.T) {
	r := NewRoom("ABC123", time.Second, 20, nil)
	r.players["1"] = &Player{ID: "1", X: 99, Y: 99}
	r.status = "playing"
	r.ApplyInput("1", Input{DX: 10, DY: 10, Sequence: 1})
	r.step(10)
	if r.players["1"].X > ArenaWidth || r.players["1"].Y > ArenaHeight { t.Fatal("player escaped arena") }
}
