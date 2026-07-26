package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/raxima/seatpicker/internal/model"
	"github.com/raxima/seatpicker/internal/service"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
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
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid login, password or role"})
		case errors.Is(err, model.ErrAlreadyExists):
			c.JSON(http.StatusConflict, gin.H{"error": "login already taken"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	c.JSON(http.StatusCreated, registerResponse{
		ID:    u.ID,
		Login: u.Login,
		Role:  u.Role,
	})
}
