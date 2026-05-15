package main

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/eliau2005/openadsource/server/internal/capping"
	"github.com/eliau2005/openadsource/server/internal/config"
	"github.com/eliau2005/openadsource/server/internal/db"
	"github.com/eliau2005/openadsource/server/internal/delivery"
	"github.com/eliau2005/openadsource/server/internal/httpmw"
	"github.com/eliau2005/openadsource/server/internal/registry"
	"github.com/eliau2005/openadsource/server/internal/storage"
	"github.com/eliau2005/openadsource/server/internal/targeting"
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

	// --- storage ---
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

	// --- targeting extractors ---
	ipResolver, err := targeting.NewIPResolver(cfg.TrustedProxies)
	if err != nil {
		log.Fatal().Err(err).Msg("trusted proxies config invalid")
	}
	geoResolver, err := targeting.NewGeoResolver(cfg.GeoIPDBPath)
	if err != nil {
		log.Fatal().Err(err).Msg("geoip resolver init failed")
	}
	defer geoResolver.Close()

	// --- redis + budget enforcer (optional in dev) ---
	var redisClient *redis.Client
	if cfg.RedisURL != "" {
		opt, err := redis.ParseURL(cfg.RedisURL)
		if err != nil {
			log.Fatal().Err(err).Msg("REDIS_URL parse failed")
		}
		redisClient = redis.NewClient(opt)
		if err := redisClient.Ping(ctx).Err(); err != nil {
			log.Fatal().Err(err).Msg("redis unreachable")
		}
		log.Info().Msg("redis connected")
		defer redisClient.Close()
	} else {
		log.Info().Msg("redis not configured; budget enforcement will be a no-op stub")
	}
	budget, err := capping.New(ctx, redisClient)
	if err != nil {
		log.Fatal().Err(err).Msg("budget enforcer init failed")
	}
	freq, err := capping.NewFrequencyEnforcer(ctx, redisClient)
	if err != nil {
		log.Fatal().Err(err).Msg("freq enforcer init failed")
	}

	// --- tracking signer (Phase 4) ---
	if len(cfg.TrackingSecret) < 8 {
		log.Fatal().Msg("TRACKING_SECRET must be at least 8 chars")
	}
	signer := tracking.NewSigner(cfg.TrackingSecret, cfg.TrackingTokenTTL)
	trackHandler := tracking.NewHandler(signer, redisClient)

	// --- registry refresher: load first snapshot synchronously, then run ---
	reg := registry.New(pool, cfg.RegistryRefreshInterval, redisClient)
	refresherDone := make(chan error, 1)
	go func() { refresherDone <- reg.Run(ctx) }()
	if err := reg.WaitReady(ctx); err != nil {
		log.Fatal().Err(err).Msg("registry never became ready")
	}

	// --- delivery handler ---
	deliveryHandler := delivery.New(cfg, reg, resolver, budget, freq, ipResolver, geoResolver, signer)

	// --- router ---
	var healthReady atomic.Bool
	healthReady.Store(true)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		if !healthReady.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("registry not ready"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Group(func(r chi.Router) {
		r.Use(httpmw.CORS("*"))
		r.Get("/vast", deliveryHandler.ServeVAST)
		r.Get("/track", trackHandler.ServeTrack)
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
	if err := <-refresherDone; err != nil && !errors.Is(err, context.Canceled) {
		log.Error().Err(err).Msg("refresher exited with error")
	}
	log.Info().Msg("adserver stopped")
}
