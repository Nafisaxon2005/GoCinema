package seats

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// POST /shows/:id/seats/:seatId/book
func (h *Handler) Book(c *gin.Context) {
	showID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid show id"})
		return
	}
	seatID, err := strconv.ParseInt(c.Param("seatId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid seat id"})
		return
	}

	userIDVal, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := userIDVal.(int64)

	bookingID, err := h.repo.BookSeat(c.Request.Context(), showID, seatID, userID)
	switch {
	case err == nil:
		c.JSON(http.StatusCreated, gin.H{"bookingId": bookingID})
	case errors.Is(err, ErrSeatTaken):
		c.JSON(http.StatusConflict, gin.H{"error": "ErrSeatTaken"})
	case errors.Is(err, ErrShowNotAvailable):
		c.JSON(http.StatusBadRequest, gin.H{"error": "ErrShowNotAvailable"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}
