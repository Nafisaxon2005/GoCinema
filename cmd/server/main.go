package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/raxima/seatpicker/internal/router"
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
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

	r := router.New(pool)

	port := getenv("APP_PORT", "8080")
	log.Printf("сервер запущен на :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("сервер упал: %v", err)
	}
}
