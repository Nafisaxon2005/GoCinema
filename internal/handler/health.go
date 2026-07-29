package handler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/raxima/seatpicker/internal/middleware"
)

type HealthHandler struct {
	DB     *pgxpool.Pool
	logger *slog.Logger
}

func NewHealthHandler(db *pgxpool.Pool, logger *slog.Logger) *HealthHandler {
	return &HealthHandler{DB: db, logger: logger}
}

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
