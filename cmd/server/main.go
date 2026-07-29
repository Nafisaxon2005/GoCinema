package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/raxima/seatpicker/internal/router"
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvDuration(key, fallback string) time.Duration {
	d, err := time.ParseDuration(getenv(key, fallback))
	if err != nil {
		log.Fatalf("неверный формат длительности для %s: %v", key, err)
	}
	return d
}

func main() {
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
		log.Fatalf("не удалось создать пул подключений к БД: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("не удалось подключиться к БД: %v", err)
	}
	log.Println("подключение к БД установлено")

	cfg := router.Config{
		JWTSecret:  []byte(getenv("JWT_SECRET", "dev-secret-change-me")),
		AccessTTL:  getenvDuration("ACCESS_TOKEN_TTL", "15m"),
		RefreshTTL: getenvDuration("REFRESH_TOKEN_TTL", "720h"), // 30 дней
	}

	r := router.New(pool, cfg)

	port := getenv("APP_PORT", "8080")
	log.Printf("сервер запущен на :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("сервер упал: %v", err)
	}
}
