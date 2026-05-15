package main

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/eliau2005/openadsource/server/internal/config"
	"github.com/eliau2005/openadsource/server/internal/db"
	"github.com/eliau2005/openadsource/server/internal/delivery"
	"github.com/eliau2005/openadsource/server/internal/httpmw"
	"github.com/eliau2005/openadsource/server/internal/storage"
	"github.com/eliau2005/openadsource/server/internal/tracking"
)

func main() {
	cfg := config.Load()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if lvl, err := zerolog.ParseLevel(cfg.LogLevel); err == nil {
		zerolog.SetGlobalLevel(lvl)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("db pool init failed")
	}
	defer pool.Close()

	if err := db.PingWithRetry(ctx, pool, 15, 2*time.Second); err != nil {
		log.Fatal().Err(err).Msg("postgres unreachable")
	}
	log.Info().Msg("postgres connected")

	queries := db.New(pool)

	var s3Client *storage.S3Client
	if cfg.S3Configured() {
		s3Client, err = storage.NewS3Client(ctx, cfg)
		if err != nil {
			log.Fatal().Err(err).Msg("s3 client init failed")
		}
		log.Info().Str("bucket", s3Client.Bucket()).Msg("s3 client ready")
	} else {
		log.Info().Msg("s3 not configured; internal_s3 ads will require S3_PUBLIC_BASE_URL or will error")
	}
	resolver := storage.New(cfg, s3Client)

	deliveryHandler := delivery.New(cfg, queries, resolver)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Group(func(r chi.Router) {
		r.Use(httpmw.CORS("*"))
		r.Get("/vast", deliveryHandler.ServeVAST)
		r.Get("/track", tracking.Stub)
		// OPTIONS handlers are required so chi routes preflight requests
		// through the CORS middleware (which short-circuits with 204).
		// Without them, chi returns 405 before the middleware runs.
		preflight := func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}
		r.Options("/vast", preflight)
		r.Options("/track", preflight)
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	idleConnsClosed := make(chan struct{})
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("shutdown error")
		}
		close(idleConnsClosed)
	}()

	log.Info().Str("addr", srv.Addr).Msg("adserver listening")
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal().Err(err).Msg("listen failed")
	}
	<-idleConnsClosed
	log.Info().Msg("adserver stopped")
}
