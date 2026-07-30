package router

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/raxima/seatpicker/internal/handler"
	"github.com/raxima/seatpicker/internal/middleware"
	"github.com/raxima/seatpicker/internal/repository"
	"github.com/raxima/seatpicker/internal/service"
)

// DB объединяет то, что нужно репозиториям (DBTX), и то, что нужно
// health-проверке (Pinger). *pgxpool.Pool реализует оба интерфейса,
// как и pgxmock.PgxPoolIface — в тестах.
type DB interface {
	repository.DBTX
	handler.Pinger
}

func New(db DB, logger *slog.Logger, authCfg service.AuthConfig) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	r.Use(middleware.RequestID())
	r.Use(middleware.Logger(logger))

	healthHandler := handler.NewHealthHandler(db, logger)
	r.GET("/health", healthHandler.Health)

	userRepo := repository.NewPgUserRepo(db, logger)
	refreshTokenRepo := repository.NewPgRefreshTokenRepo(db)
	authService := service.NewAuthService(userRepo, refreshTokenRepo, authCfg, logger)
	authHandler := handler.NewAuthHandler(authService, logger)

	auth := r.Group("/auth")
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)
	auth.POST("/refresh", authHandler.Refresh)
	auth.POST("/logout", authHandler.Logout)

	showRepo := repository.NewPgShowRepo(db)
	showService := service.NewShowService(showRepo)
	showHandler := handler.NewShowHandler(showService)

	shows := r.Group("/shows")
	shows.GET("", showHandler.List)
	shows.GET("/:id", showHandler.GetByID)

	// TODO: сюда подключаются роуты дорожек A/B/C.

	return r
}
