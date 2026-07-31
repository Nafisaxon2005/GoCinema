package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/raxima/seatpicker/internal/router"
	"github.com/raxima/seatpicker/internal/service"
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvDuration(logger *slog.Logger, key, fallback string) time.Duration {
	d, err := time.ParseDuration(getenv(key, fallback))
	if err != nil {
		logger.Error("неверный формат длительности", "key", key, "error", err)
		os.Exit(1)
	}
	return d
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	ctx := context.Background()

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		getenv("DB_USER", "seatpicker"),
		getenv("DB_PASSWORD", "seatpicker"),
		getenv("DB_HOST", "db"),
		getenv("DB_PORT", "5432"),
		getenv("DB_NAME", "seatpicker"),
	)

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		logger.Error("не удалось создать пул подключений к БД", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		logger.Error("не удалось подключиться к БД", "error", err)
		os.Exit(1)
	}
	logger.Info("подключение к БД установлено")

	authCfg := service.AuthConfig{
		JWTSecret:  []byte(getenv("JWT_SECRET", "dev-secret-change-me")),
		AccessTTL:  getenvDuration(logger, "JWT_ACCESS_TTL", "15m"),
		RefreshTTL: getenvDuration(logger, "JWT_REFRESH_TTL", "720h"),
	}

	r := router.New(pool, logger, authCfg)

	port := getenv("APP_PORT", "8080")
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("сервер запущен", "port", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("сервер упал", "error", err)
			os.Exit(1)
		}
	}()

	stopCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-stopCtx.Done()

	logger.Info("получен сигнал завершения, начинаю graceful shutdown")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("ошибка при остановке HTTP-сервера", "error", err)
	} else {
		logger.Info("HTTP-сервер остановлен")
	}

	pool.Close()
	logger.Info("пул подключений к БД закрыт")

	logger.Info("graceful shutdown завершён")
}
