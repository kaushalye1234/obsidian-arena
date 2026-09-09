package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kaushalye1234/obsidian-arena/internal/api"
	"github.com/kaushalye1234/obsidian-arena/internal/config"
	"github.com/kaushalye1234/obsidian-arena/internal/game"
	"github.com/kaushalye1234/obsidian-arena/internal/store"
)

func main() {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM); defer stop()
	dataStore, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil { slog.Error("database connection failed", "error", err); os.Exit(1) }
	defer dataStore.Close()
	hub := game.NewHub(cfg.MatchDuration, cfg.TickRate, func(roomCode string, result game.MatchResult) {
		persistCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second); defer cancel()
		if err := dataStore.SaveMatch(persistCtx, result.MatchID, roomCode, result.Started, result.Scores); err != nil { slog.Error("match persistence failed", "matchId", result.MatchID, "error", err) }
	})
	handler := api.New(dataStore, hub, cfg.Origins).Routes()
	server := &http.Server{Addr: ":"+cfg.Port, Handler: handler, ReadHeaderTimeout: 5*time.Second, IdleTimeout: 60*time.Second}
	go func() { slog.Info("server started", "port", cfg.Port); if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) { slog.Error("server failed", "error", err); stop() } }()
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second); defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil { slog.Error("shutdown failed", "error", err) }
}
