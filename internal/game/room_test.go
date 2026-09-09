package game

import (
	"testing"
	"time"
)

type fakeClient struct{ events []Event }

func (f *fakeClient) Send(e Event) bool {
	f.events = append(f.events, e)
	return true
}

func TestRoomRequiresTwoReadyPlayers(t *testing.T) {
	r := NewRoom("ABC123", time.Second, 20, nil)
	r.AddPlayer("1", "Ada", &fakeClient{})
	if r.CanStart() {
		t.Fatal("one player must not start a match")
	}
	r.AddPlayer("2", "Linus", &fakeClient{})
	r.SetReady("1", true)
	r.SetReady("2", true)
	if !r.CanStart() {
		t.Fatal("two ready players should start")
	}
}

func TestMovementIsClampedToArena(t *testing.T) {
	r := NewRoom("ABC123", time.Second, 20, nil)
	r.players["1"] = &Player{ID: "1", X: 99, Y: 99}
	r.status = "playing"
	r.ApplyInput("1", Input{DX: 10, DY: 10, Sequence: 1})
	r.step(10)
	if r.players["1"].X > ArenaWidth || r.players["1"].Y > ArenaHeight {
		t.Fatal("player escaped arena")
	}
}

func TestCrystalCollectionAwardsScoreAndRespawns(t *testing.T) {
	r := NewRoom("ABC123", time.Second, 20, nil)
	r.players["1"] = &Player{ID: "1", X: 25, Y: 25}
	r.crystals = []Crystal{{ID: "crystal-0", X: 25, Y: 25, Value: CrystalScore}}
	r.status = "playing"

	r.step(0)

	if r.players["1"].Score != CrystalScore {
		t.Fatalf("expected score %d, got %d", CrystalScore, r.players["1"].Score)
	}
	if r.crystals[0].X == 25 && r.crystals[0].Y == 25 {
		t.Fatal("collected crystal must respawn")
	}
}

func TestCrystalSpawnIsDeterministicForRoomCode(t *testing.T) {
	first := NewRoom("ABC123", time.Second, 20, nil)
	second := NewRoom("ABC123", time.Second, 20, nil)
	first.spawnCrystalsLocked()
	second.spawnCrystalsLocked()

	for index := range first.crystals {
		if first.crystals[index] != second.crystals[index] {
			t.Fatalf("crystal %d differs for the same room seed", index)
		}
	}
}
