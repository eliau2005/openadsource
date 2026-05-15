package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/eliau2005/openadsource/server/internal/config"
)

func main() {
	cfg := config.Load()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if lvl, err := zerolog.ParseLevel(cfg.LogLevel); err == nil {
		zerolog.SetGlobalLevel(lvl)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Info().Msg("worker placeholder started — Phase 4 will implement the Redis->Postgres reconciler")
	<-ctx.Done()
	log.Info().Msg("worker stopped")
	_ = os.Stdout.Sync()
}
