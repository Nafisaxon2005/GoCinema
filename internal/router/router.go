package router

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/raxima/seatpicker/internal/middleware"

	"github.com/raxima/seatpicker/internal/handler"
	"github.com/raxima/seatpicker/internal/repository"
	"github.com/raxima/seatpicker/internal/service"
)

// Config — настройки, нужные роутеру для сборки JWT-зависимых компонентов.
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

	// TODO: сюда подключаются роуты дорожек A/B/C.
	_ = middleware.Auth
	return r
}
