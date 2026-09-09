package game

import (
	"testing"
	"time"
)

func TestLobbySnapshotsIncludeRosterAndReadiness(t *testing.T) {
	r := NewRoom("LOBBY1", time.Minute, 20, nil)
	first, second := &fakeClient{}, &fakeClient{}
	r.AddPlayer("1", "Ada", first)
	r.AddPlayer("2", "Linus", second)
	r.SetReady("1", true)
	for _, client := range []*fakeClient{first, second} {
		event := client.events[len(client.events)-1]
		snapshot, ok := event.Data.(Snapshot)
		if !ok || event.Type != "snapshot" {
			t.Fatal("expected a lobby snapshot")
		}
		if len(snapshot.Players) != 2 || !snapshot.Players["1"].Ready || snapshot.RemainingMS != 60000 {
			t.Fatal("lobby snapshot must include both players, readiness and full duration")
		}
	}
}

func TestFinishedRoomDoesNotMoveOrScoreAndCompletesOnce(t *testing.T) {
	completed := 0
	r := NewRoom("FINISH", time.Minute, 20, func(MatchResult) { completed++ })
	r.AddPlayer("1", "Ada", &fakeClient{})
	r.status = "playing"
	r.ApplyInput("1", Input{DX: 1, Sequence: 1})
	r.finish()
	before := *r.players["1"]
	r.crystals = []Crystal{{ID: "c1", X: before.X, Y: before.Y, Value: 10}}
	r.ApplyInput("1", Input{DX: 1, Sequence: 2})
	r.step(1)
	r.finish()
	if *r.players["1"] != before || completed != 1 {
		t.Fatal("finished rooms must freeze and persist their result only once")
	}
}
