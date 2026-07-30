package router

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/raxima/seatpicker/internal/middleware"
	"github.com/raxima/seatpicker/internal/seats"

	"github.com/raxima/seatpicker/internal/handler"
	"github.com/raxima/seatpicker/internal/repository"
	"github.com/raxima/seatpicker/internal/service"
)

type Config struct {
	JWTSecret  []byte
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

func New(db *pgxpool.Pool, cfg Config) *gin.Engine {
	r := gin.Default()

	healthHandler := handler.NewHealthHandler(db)
	r.GET("/health", healthHandler.Health)

	userRepo := repository.NewPgUserRepo(db)
	refreshTokenRepo := repository.NewPgRefreshTokenRepo(db)
	authService := service.NewAuthService(userRepo, refreshTokenRepo, service.AuthConfig{
		JWTSecret:  cfg.JWTSecret,
		AccessTTL:  cfg.AccessTTL,
		RefreshTTL: cfg.RefreshTTL,
	})
	authHandler := handler.NewAuthHandler(authService)

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
	seatsRepo := seats.NewRepository(db)
	seatsHandler := seats.NewHandler(seatsRepo)

	protected := r.Group("/")
	protected.Use(middleware.Auth(cfg.JWTSecret))

	protected.POST("/shows/:id/seats/:seatId/book", seatsHandler.Book)
	protected.DELETE("/bookings/:bookingId", seatsHandler.Cancel)

	return r
}
