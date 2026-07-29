package router

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/raxima/seatpicker/internal/middleware"

	"github.com/raxima/seatpicker/internal/handler"
	"github.com/raxima/seatpicker/internal/middleware"
	"github.com/raxima/seatpicker/internal/repository"
	"github.com/raxima/seatpicker/internal/service"
)

func New(db *pgxpool.Pool, logger *slog.Logger) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	r.Use(middleware.RequestID())
	r.Use(middleware.Logger(logger))

	healthHandler := handler.NewHealthHandler(db, logger)
	r.GET("/health", healthHandler.Health)

	userRepo := repository.NewPgUserRepo(db, logger)
	authService := service.NewAuthService(userRepo, logger)
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
	_ = middleware.Auth
	return r
}
