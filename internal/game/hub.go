package game

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"math/big"
	"sync"
	"time"
)

const roomAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

type Hub struct {
	mu sync.RWMutex
	rooms map[string]*Room
	duration time.Duration
	tickRate int
	onComplete func(string, MatchResult)
}

func NewHub(duration time.Duration, tickRate int, onComplete func(string, MatchResult)) *Hub {
	return &Hub{rooms: map[string]*Room{}, duration: duration, tickRate: tickRate, onComplete: onComplete}
}

func (h *Hub) Create() *Room {
	h.mu.Lock(); defer h.mu.Unlock()
	for {
		code := roomCode(6)
		if h.rooms[code] == nil {
			room := NewRoom(code, h.duration, h.tickRate, func(result MatchResult) { if h.onComplete != nil { h.onComplete(code, result) } })
			h.rooms[code] = room; return room
		}
	}
}

func (h *Hub) Get(code string) (*Room, error) {
	h.mu.RLock(); defer h.mu.RUnlock()
	room := h.rooms[code]
	if room == nil { return nil, errors.New("room not found") }
	return room, nil
}

func roomCode(length int) string {
	result := make([]byte, length)
	for i := range result { n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(roomAlphabet)))); result[i] = roomAlphabet[n.Int64()] }
	return string(result)
}

func newID() string { b := make([]byte, 16); _, _ = rand.Read(b); return hex.EncodeToString(b) }
