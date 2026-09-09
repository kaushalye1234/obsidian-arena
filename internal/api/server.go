package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/kaushalye1234/obsidian-arena/internal/game"
	"github.com/kaushalye1234/obsidian-arena/internal/store"
)

type Server struct {
	store *store.Store
	hub *game.Hub
	origins map[string]bool
}

func New(dataStore *store.Store, hub *game.Hub, origins []string) *Server {
	allowed := make(map[string]bool, len(origins))
	for _, origin := range origins { allowed[origin] = true }
	return &Server{store: dataStore, hub: hub, origins: allowed}
}

func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(s.cors, requestLog, recoverer)
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) { respond(w, http.StatusOK, map[string]string{"status":"ok"}) })
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/players", s.createPlayer)
		r.Get("/leaderboard", s.leaderboard)
		r.Get("/players/{playerID}/progress", s.getProgress)
		r.Put("/players/{playerID}/progress", s.saveProgress)
		r.Post("/rooms", s.createRoom)
		r.Get("/rooms/{code}/connect", s.connectRoom)
	})
	return r
}

func (s *Server) createPlayer(w http.ResponseWriter, r *http.Request) {
	var body struct { DisplayName string `json:"displayName"` }
	if json.NewDecoder(r.Body).Decode(&body) != nil { problem(w, http.StatusBadRequest, "invalid JSON"); return }
	body.DisplayName = strings.TrimSpace(body.DisplayName)
	if len(body.DisplayName) < 2 || len(body.DisplayName) > 24 { problem(w, http.StatusBadRequest, "displayName must contain 2-24 characters"); return }
	id, err := s.store.EnsurePlayer(r.Context(), body.DisplayName)
	if err != nil { problem(w, http.StatusInternalServerError, "could not create player"); return }
	respond(w, http.StatusCreated, map[string]string{"playerId": id, "displayName": body.DisplayName})
}

func (s *Server) leaderboard(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if parsed, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && parsed > 0 && parsed <= 100 { limit = parsed }
	entries, err := s.store.Leaderboard(r.Context(), limit)
	if err != nil { problem(w, http.StatusInternalServerError, "could not load leaderboard"); return }
	respond(w, http.StatusOK, map[string]any{"entries": entries})
}

func (s *Server) getProgress(w http.ResponseWriter, r *http.Request) {
	progress, err := s.store.Progress(r.Context(), chi.URLParam(r, "playerID"))
	if err != nil { problem(w, http.StatusNotFound, "player not found"); return }
	respond(w, http.StatusOK, map[string]any{"progress": progress})
}

func (s *Server) saveProgress(w http.ResponseWriter, r *http.Request) {
	var body struct { Progress json.RawMessage `json:"progress"` }
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&body) != nil || !json.Valid(body.Progress) { problem(w, http.StatusBadRequest, "progress must be valid JSON"); return }
	if err := s.store.SaveProgress(r.Context(), chi.URLParam(r, "playerID"), body.Progress); err != nil { problem(w, http.StatusInternalServerError, "could not save progress"); return }
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) createRoom(w http.ResponseWriter, _ *http.Request) {
	room := s.hub.Create()
	respond(w, http.StatusCreated, map[string]string{"roomCode": room.Code})
}

type clientMessage struct {
	Type string `json:"type"`
	Ready bool `json:"ready,omitempty"`
	DX float64 `json:"dx,omitempty"`
	DY float64 `json:"dy,omitempty"`
	Sequence uint64 `json:"sequence,omitempty"`
}

func (s *Server) connectRoom(w http.ResponseWriter, r *http.Request) {
	if !s.originAllowed(r.Header.Get("Origin")) { problem(w, http.StatusForbidden, "origin not allowed"); return }
	room, err := s.hub.Get(strings.ToUpper(chi.URLParam(r, "code")))
	if err != nil { problem(w, http.StatusNotFound, "room not found"); return }
	playerID, name := r.URL.Query().Get("playerId"), strings.TrimSpace(r.URL.Query().Get("name"))
	if playerID == "" || name == "" { problem(w, http.StatusBadRequest, "playerId and name are required"); return }
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil { return }
	defer conn.CloseNow()
	client := newWSClient(conn)
	if !room.AddPlayer(playerID, name, client) { conn.Close(websocket.StatusPolicyViolation, "room unavailable or full"); return }
	defer room.RemovePlayer(playerID)
	ctx, cancel := context.WithCancel(r.Context()); defer cancel()
	go client.writeLoop(ctx)
	client.Send(game.Event{Type: "connected", Data: map[string]string{"roomCode": room.Code, "playerId": playerID}})
	for {
		_, payload, err := conn.Read(ctx)
		if err != nil { return }
		var message clientMessage
		if json.Unmarshal(payload, &message) != nil { client.Send(game.Event{Type:"error", Data:"invalid message"}); continue }
		switch message.Type {
		case "ready": room.SetReady(playerID, message.Ready)
		case "start": if !room.Start() { client.Send(game.Event{Type:"error", Data:"at least two players must be ready"}) }
		case "input": room.ApplyInput(playerID, game.Input{DX: message.DX, DY: message.DY, Sequence: message.Sequence})
		default: client.Send(game.Event{Type:"error", Data:"unknown message type"})
		}
	}
}

func (s *Server) originAllowed(origin string) bool { return origin == "" || s.origins[origin] }

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && s.origins[origin] { w.Header().Set("Access-Control-Allow-Origin", origin); w.Header().Set("Vary", "Origin"); w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization"); w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS") }
		if r.Method == http.MethodOptions { if !s.originAllowed(origin) { problem(w, http.StatusForbidden, "origin not allowed"); return }; w.WriteHeader(http.StatusNoContent); return }
		next.ServeHTTP(w, r)
	})
}

func requestLog(next http.Handler) http.Handler { return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { started := time.Now(); next.ServeHTTP(w,r); slog.Info("request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(started)) }) }
func recoverer(next http.Handler) http.Handler { return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { defer func(){ if value := recover(); value != nil { slog.Error("panic", "error", value); problem(w, http.StatusInternalServerError, "internal server error") } }(); next.ServeHTTP(w,r) }) }
func respond(w http.ResponseWriter, status int, value any) { w.Header().Set("Content-Type", "application/json"); w.WriteHeader(status); _ = json.NewEncoder(w).Encode(value) }
func problem(w http.ResponseWriter, status int, message string) { respond(w, status, map[string]any{"status":status,"error":message}) }
