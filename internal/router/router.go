package router

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/raxima/seatpicker/internal/handler"
	"github.com/raxima/seatpicker/internal/repository"
	"github.com/raxima/seatpicker/internal/service"
)

func New(db *pgxpool.Pool) *gin.Engine {
	r := gin.Default()

	healthHandler := handler.NewHealthHandler(db)
	r.GET("/health", healthHandler.Health)

	userRepo := repository.NewPgUserRepo(db)
	authService := service.NewAuthService(userRepo)
	authHandler := handler.NewAuthHandler(authService)

	auth := r.Group("/auth")
	auth.POST("/register", authHandler.Register)
	// TODO: J-04 — auth.POST("/login", authHandler.Login)
	//Nafisa's part

	// TODO: сюда подключаются роуты дорожек A/B/C.

	return r
}
