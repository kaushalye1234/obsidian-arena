package api

import (
	"context"
	"encoding/json"
	"time"

	"github.com/coder/websocket"
	"github.com/kaushalye1234/obsidian-arena/internal/game"
)

type wsClient struct {
	conn *websocket.Conn
	send chan game.Event
}

func newWSClient(conn *websocket.Conn) *wsClient { return &wsClient{conn: conn, send: make(chan game.Event, 64)} }

func (c *wsClient) Send(event game.Event) bool {
	select { case c.send <- event: return true; default: return false }
}

func (c *wsClient) writeLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done(): return
		case event := <-c.send:
			writeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			err := wsjsonWrite(writeCtx, c.conn, event); cancel()
			if err != nil { return }
		}
	}
}

func wsjsonWrite(ctx context.Context, conn *websocket.Conn, value any) error {
	data, err := json.Marshal(value)
	if err != nil { return err }
	return conn.Write(ctx, websocket.MessageText, data)
}
