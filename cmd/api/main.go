package api

import (
	"context"
	"errors"
	"fmt"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"kabsa/internal/app/user"
	"kabsa/internal/cache"
	"kabsa/internal/config"
	"kabsa/internal/db"
	"kabsa/internal/db/repository"
	"kabsa/internal/http/handlers/health"
	userhandler "kabsa/internal/http/handlers/user"
	"kabsa/internal/http/router"
	"kabsa/internal/kafka"
	"kabsa/internal/logging"
	"kabsa/internal/telemetry"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Top-level context with graceful shutdown on SIGINT/SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// 1) Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// 2) Initialize logger
	logger := logging.New(
		cfg.Observability.ServiceName,
		cfg.Observability.ServiceEnv,
	)

	logger.Info("starting service",
		"env", cfg.Environment,
	)

	// 3) Initialize telemetry (OpenTelemetry)
	otelShutdown, err := telemetry.Setup(ctx, cfg.Observability, logger)
	if err != nil {
		logger.Error("failed to setup telemetry", "error", err)
		os.Exit(1)
	}
	// ensure we flush / shut down exporter on exit
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := otelShutdown(shutdownCtx); err != nil {
			logger.Error("failed to shutdown telemetry", "error", err)
		}
	}()

	// 4) Initialize Postgres (Ent client)
	dbClient, err := db.NewClient(ctx, cfg.Postgres, logger)
	if err != nil {
		logger.Error("failed to init database", "error", err)
		os.Exit(1)
	}

	defer func(dbClient *db.Client) {
		_ = dbClient.Close()
	}(dbClient)

	// 5) Initialize Redis
	redisClient, err := cache.NewRedisClient(ctx, cfg.Redis, logger)
	if err != nil {
		logger.Error("failed to init redis", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			logger.Error("failed to close redis", "error", err)
		}
	}()

	// 6) Initialize Kafka bus (Watermill)
	bus, closeBus, err := kafka.NewBus(cfg.Kafka, logger)
	if err != nil {
		logger.Error("failed to init kafka bus", "error", err)
		os.Exit(1)
	}
	defer func() {
		_ = closeBus(context.Background())
	}()

	// 7) Kafka router (for consumers)
	kafkaRouter, err := kafka.NewRouter(ctx, cfg.Kafka, logger)
	if err != nil {
		logger.Error("failed to init kafka router", "error", err)
		os.Exit(1)
	}

	// 8) Construct repositories & services
	userRepo := repository.NewUserRepository(dbClient, logger)
	userCache := cache.NewUserCache(redisClient)
	userEvents := kafka.NewUserEvents(bus, cfg.Kafka, logger)

	userService := user.NewService(
		userRepo,
		userCache,
		dbClient,   // db.Transactor
		userEvents, // app/user.Events
		logger)

	// 8) HTTP handlers
	healthHandler := health.NewHandler(dbClient, redisClient)
	userHandler := userhandler.NewHandler(userService, logger)

	// 9) HTTP router
	httpRouter := router.NewRouter(
		logger,
		cfg.Observability.ServiceName,
		healthHandler,
		userHandler,
	)

	// 10) HTTP server
	srv := &http.Server{
		Addr: fmt.Sprintf("%s:%d", cfg.HTTP.Host, cfg.HTTP.Port),
		Handler: otelhttp.NewHandler(
			httpRouter,
			cfg.Observability.ServiceName, // span name prefix
		),
	}

	// 11) Start concurrent processes (HTTP server, Kafka router)
	errCh := make(chan error, 2)

	go func() {
		logger.Info("http server starting",
			"host", cfg.HTTP.Host,
			"port", cfg.HTTP.Port,
		)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	go func() {
		logger.Info("kafka router starting")
		if err := kafkaRouter.Run(ctx); err != nil {
			errCh <- err
		}
	}()

	// 12) Wait for shutdown signal or an error
	select {
	case <-ctx.Done():
		logger.Info("received shutdown signal")
	case err := <-errCh:
		logger.Error("fatal error from subsystem", "error", err)
		// Cancel context to trigger shutdown of others
		stop()
	}

	// 13) Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("failed to shutdown http server", "error", err)
	}

	logger.Info("service stopped")
}
