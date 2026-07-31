package router

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"github.com/raxima/seatpicker/internal/handler"
	"github.com/raxima/seatpicker/internal/middleware"
	"github.com/raxima/seatpicker/internal/model"
	"github.com/raxima/seatpicker/internal/repository"
	"github.com/raxima/seatpicker/internal/seats"
	"github.com/raxima/seatpicker/internal/service"
)

type DB interface {
	repository.DBTX
	handler.Pinger
	// Begin нужен для seats.Repository (RefundBooking работает в транзакции).
	// Реальный *pgxpool.Pool, который передаётся сюда из main.go, этот метод уже реализует.
	Begin(ctx context.Context) (pgx.Tx, error)
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
	shows.GET("/:id/poster", showHandler.GetPoster)

	protected := shows.Group("", middleware.AuthMiddleware(authCfg.JWTSecret), middleware.RequireRole(model.RoleOrganizer, model.RoleAdmin))
	protected.POST("", showHandler.Create)
	protected.PUT("/:id", showHandler.Update)
	protected.DELETE("/:id", showHandler.Delete)
	protected.POST("/:id/poster", showHandler.UploadPoster)
	protected.POST("/:id/seatmap", showHandler.GenerateSeatMap)
	protected.GET("/:id/stats", showHandler.GetStats)
	protected.PUT("/:id/cancel", showHandler.Cancel)

	adminRepo := repository.NewPgAdminRepo(db)
	seatsRepo := seats.NewRepository(db)
	adminService := service.NewAdminService(adminRepo, showRepo, seatsRepo)
	adminHandler := handler.NewAdminHandler(adminService)

	admin := r.Group("/admin", middleware.AuthMiddleware(authCfg.JWTSecret), middleware.RequireRole(model.RoleAdmin))
	admin.GET("/stats", adminHandler.GetStats)
	admin.GET("/shows", adminHandler.GetAllShows)
	admin.PUT("/shows/:id/moderate", adminHandler.ModerateShow)
	admin.GET("/refunds", adminHandler.ListRefunds)
	admin.GET("/bookings/:id/refund", adminHandler.RefundBooking)
	admin.GET("/export", adminHandler.ExportReport)
	admin.GET("/holds", adminHandler.ListHolds)

	return r
}
