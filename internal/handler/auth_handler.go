package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/raxima/seatpicker/internal/httpx"

	"github.com/raxima/seatpicker/internal/middleware"
	"github.com/raxima/seatpicker/internal/model"
	"github.com/raxima/seatpicker/internal/service"
)

type AuthHandler struct {
	auth   *service.AuthService
	logger *slog.Logger
}

func NewAuthHandler(auth *service.AuthService, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{auth: auth, logger: logger}
}

type registerRequest struct {
	Login    string     `json:"login" binding:"required"`
	Password string     `json:"password" binding:"required"`
	Role     model.Role `json:"role" binding:"required"`
}

type registerResponse struct {
	ID    int64      `json:"id"`
	Login string     `json:"login"`
	Role  model.Role `json:"role"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	log := h.logger.With(
		"layer", "handler",
		"op", "Register",
		"request_id", middleware.RequestIDFromContext(c.Request.Context()),
	)

	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("невалидное тело запроса", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	u, err := h.auth.Register(c.Request.Context(), service.RegisterInput{
		Login:    req.Login,
		Password: req.Password,
		Role:     req.Role,
	})
	if err != nil {
		switch {
		case errors.Is(err, model.ErrInvalid):
			log.Warn("ошибка валидации при регистрации", "login", req.Login, "error", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid login, password or role"})
		case errors.Is(err, model.ErrAlreadyExists):
			log.Warn("логин уже занят", "login", req.Login)
			c.JSON(http.StatusConflict, gin.H{"error": "login already taken"})
		default:
			log.Error("внутренняя ошибка при регистрации", "login", req.Login, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	log.Info("запрос на регистрацию обработан", "user_id", u.ID, "login", u.Login)
	c.JSON(http.StatusCreated, registerResponse{
		ID:    u.ID,
		Login: u.Login,
		Role:  u.Role,
	})
}

type loginRequest struct {
	Login    string `json:"login" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type tokenPairResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.RespondError(c, model.ErrInvalid)
		return
	}

	pair, err := h.auth.Login(c.Request.Context(), req.Login, req.Password)
	if err != nil {
		httpx.RespondError(c, err)
		return
	}

	httpx.RespondOK(c, tokenPairResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
	})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.RespondError(c, model.ErrInvalid)
		return
	}

	pair, err := h.auth.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		httpx.RespondError(c, err)
		return
	}

	httpx.RespondOK(c, tokenPairResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
	})
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var req logoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.RespondError(c, model.ErrInvalid)
		return
	}

	if err := h.auth.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		httpx.RespondError(c, err)
		return
	}

	httpx.RespondOK(c, gin.H{"status": "ok"})
}
