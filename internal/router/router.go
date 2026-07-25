package router

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/raxima/seatpicker/internal/handler"
)

// New собирает gin.Engine. Каждая дорожка добавляет сюда свои маршруты:
//   - J-03/J-04 (fundament): /auth/register, /auth/login
//   - A (зритель): /shows/:id/seats, /bookings
//   - B (организатор): /shows, /shows/:id/seats (bulk create)
//   - C (платформа): /analytics/...
func New(db *pgxpool.Pool) *gin.Engine {
	r := gin.Default()

	healthHandler := handler.NewHealthHandler(db)
	r.GET("/health", healthHandler.Health)

	// api := r.Group("/api")
	// TODO: сюда подключаются auth-роуты (J-03/J-04) и роуты дорожек A/B/C.

	return r
}
