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

// GetSeats handles GET /shows/:id/seats
func (h *Handler) GetSeats(c *gin.Context) {
	showID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid show id"})
		return
	}

	seats, err := h.repo.GetShowSeats(c.Request.Context(), showID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, seats)
}

// Cancel handles DELETE /bookings/:bookingId
func (h *Handler) Cancel(c *gin.Context) {
	bookingID, err := strconv.ParseInt(c.Param("bookingId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid booking id"})
		return
	}

	userIDVal, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userID := userIDVal.(int64)

	err = h.repo.CancelBooking(c.Request.Context(), bookingID, userID)
	switch {
	case err == nil:
		// 204 No Content is standard for successful DELETE operations
		c.Status(http.StatusNoContent)
	case errors.Is(err, ErrBookingNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "booking not found"})
	case errors.Is(err, ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": "ErrForbidden"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}
