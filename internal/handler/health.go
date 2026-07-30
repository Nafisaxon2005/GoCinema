package handler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/raxima/seatpicker/internal/middleware"
)

// Pinger — минимальный интерфейс для проверки доступности БД.
// *pgxpool.Pool уже реализует Ping(ctx) error, так что реальный код
// не меняется — меняется только тип поля здесь.
type Pinger interface {
	Ping(ctx context.Context) error
}

type HealthHandler struct {
	DB     Pinger
	logger *slog.Logger
}

func NewHealthHandler(db Pinger, logger *slog.Logger) *HealthHandler {
	return &HealthHandler{DB: db, logger: logger}
}

// GET /health — критерий J-01: docker compose up --build поднимает app+postgres,
// GET /health -> 200.
func (h *HealthHandler) Health(c *gin.Context) {
	log := h.logger.With(
		"layer", "handler",
		"op", "Health",
		"request_id", middleware.RequestIDFromContext(c.Request.Context()),
	)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if err := h.DB.Ping(ctx); err != nil {
		log.Error("БД недоступна", "error", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "down",
			"db":     err.Error(),
		})
		return
	}

	log.Info("health check пройден")
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
