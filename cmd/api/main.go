// @title           Subscriptions API
// @version         1.0
// @description     REST service for aggregating user subscription data
// @BasePath        /
// @schemes         http
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpSwagger "github.com/swaggo/http-swagger"
	_ "subs-service/docs"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"subs-service/internal/httpserver"
	"subs-service/internal/platform/config"
	"subs-service/internal/platform/db"
	"subs-service/internal/platform/httpmw"
	"subs-service/internal/platform/logger"
	"subs-service/internal/subscription"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	log, err := logger.New(cfg.AppEnv, cfg.LogLevel)
	if err != nil {
		panic(err)
	}
	defer func() { _ = log.Sync() }()

	log.Info("app starting",
		zap.String("app_env", cfg.AppEnv),
		zap.String("log_level", cfg.LogLevel),
		zap.String("http_port", cfg.HTTPPort),
	)

	dbCtx, cancelDB := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelDB()

	pool, err := db.NewPool(dbCtx, db.Config{
		DatabaseURL: cfg.DatabaseURL,
		MaxConns:    10,
	})
	if err != nil {
		log.Fatal("db connect failed", zap.Error(err))
	}
	defer pool.Close()

	log.Info("db connected")

	repo := subscription.NewRepositoryPG(pool)
	svc := subscription.NewService(repo)
	subHTTP := subscription.NewTransportHTTP(svc)

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)

	r.Use(httpmw.RequestLogger(log))
	r.Use(httpmw.ResponseRequestID("X-Request-ID"))
	r.Use(httpmw.Recover())
	r.Use(httpmw.AccessLog())

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Get("/swagger/*", httpSwagger.WrapHandler)

	r.Route("/api/v1/subscriptions", func(sr chi.Router) {
		sr.Mount("/", subHTTP.Routes())
	})

	srv := httpserver.New(
		r,
		httpserver.Config{
			Addr:              ":" + cfg.HTTPPort,
			ReadTimeout:       5 * time.Second,
			ReadHeaderTimeout: 5 * time.Second,
			WriteTimeout:      10 * time.Second,
			IdleTimeout:       60 * time.Second,
			MaxHeaderBytes:    1 << 20, // 1MB
		},
		log,
	)

	runCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Info("http server started", zap.String("addr", ":"+cfg.HTTPPort))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server error", zap.Error(err))
			stop()
		}
	}()

	<-runCtx.Done()
	log.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("http server shutdown failed", zap.Error(err))
	} else {
		log.Info("http server stopped")
	}
}
