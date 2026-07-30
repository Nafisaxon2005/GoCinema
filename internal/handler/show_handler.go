package handler

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/raxima/seatpicker/internal/httpx"
	"github.com/raxima/seatpicker/internal/model"
	"github.com/raxima/seatpicker/internal/service"
)

type ShowHandler struct {
	shows *service.ShowService
}

func NewShowHandler(shows *service.ShowService) *ShowHandler {
	return &ShowHandler{shows: shows}
}

const dateLayout = "2006-01-02"
const maxPosterSize = 5 * 1024 * 1024 // 5MB

func getUserID(c *gin.Context) (int64, bool) {
	val, exists := c.Get("userID")
	if !exists {
		return 0, false
	}
	id, ok := val.(int64)
	return id, ok
}

// List — GET /shows?search=&venue=&date_from=&date_to=&page=&page_size=
func (h *ShowHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))

	var dateFrom, dateTo *time.Time
	if v := c.Query("date_from"); v != "" {
		t, err := time.Parse(dateLayout, v)
		if err != nil {
			httpx.RespondError(c, model.ErrInvalid)
			return
		}
		dateFrom = &t
	}
	if v := c.Query("date_to"); v != "" {
		t, err := time.Parse(dateLayout, v)
		if err != nil {
			httpx.RespondError(c, model.ErrInvalid)
			return
		}
		t = t.Add(24 * time.Hour) // включаем весь день date_to
		dateTo = &t
	}

	resp, err := h.shows.List(c.Request.Context(), service.ListShowsInput{
		Search:   c.Query("search"),
		Venue:    c.Query("venue"),
		DateFrom: dateFrom,
		DateTo:   dateTo,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		httpx.RespondError(c, err)
		return
	}

	httpx.RespondOK(c, resp)
}

// GetByID — GET /shows/:id
func (h *ShowHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httpx.RespondError(c, model.ErrInvalid)
		return
	}

	detail, err := h.shows.GetByID(c.Request.Context(), id)
	if err != nil {
		httpx.RespondError(c, err)
		return
	}

	httpx.RespondOK(c, detail)
}

// Create — POST /shows
func (h *ShowHandler) Create(c *gin.Context) {
	organizerID, ok := getUserID(c)
	if !ok {
		httpx.RespondError(c, model.ErrUnauthorized)
		return
	}

	var in model.CreateShowInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.RespondError(c, model.ErrInvalid)
		return
	}

	show, err := h.shows.Create(c.Request.Context(), organizerID, in)
	if err != nil {
		httpx.RespondError(c, err)
		return
	}

	httpx.RespondCreated(c, show)
}

// Update — PUT /shows/:id
func (h *ShowHandler) Update(c *gin.Context) {
	organizerID, ok := getUserID(c)
	if !ok {
		httpx.RespondError(c, model.ErrUnauthorized)
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httpx.RespondError(c, model.ErrInvalid)
		return
	}

	var in model.UpdateShowInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.RespondError(c, model.ErrInvalid)
		return
	}

	show, err := h.shows.Update(c.Request.Context(), organizerID, id, in)
	if err != nil {
		httpx.RespondError(c, err)
		return
	}

	httpx.RespondOK(c, show)
}

// Delete — DELETE /shows/:id
func (h *ShowHandler) Delete(c *gin.Context) {
	organizerID, ok := getUserID(c)
	if !ok {
		httpx.RespondError(c, model.ErrUnauthorized)
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httpx.RespondError(c, model.ErrInvalid)
		return
	}

	if err := h.shows.Delete(c.Request.Context(), organizerID, id); err != nil {
		httpx.RespondError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// UploadPoster — POST /shows/:id/poster
func (h *ShowHandler) UploadPoster(c *gin.Context) {
	organizerID, ok := getUserID(c)
	if !ok {
		httpx.RespondError(c, model.ErrUnauthorized)
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httpx.RespondError(c, model.ErrInvalid)
		return
	}

	file, err := c.FormFile("poster")
	if err != nil {
		file, err = c.FormFile("file")
	}
	if err != nil {
		httpx.RespondError(c, model.ErrInvalid)
		return
	}

	if file.Size > maxPosterSize {
		httpx.RespondError(c, model.ErrInvalid)
		return
	}

	contentType := file.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		httpx.RespondError(c, model.ErrInvalid)
		return
	}

	ext := filepath.Ext(file.Filename)
	if ext == "" {
		ext = ".jpg"
	}
	filename := fmt.Sprintf("show_%d_%d%s", id, time.Now().UnixNano(), ext)
	uploadDir := "uploads/posters"
	_ = os.MkdirAll(uploadDir, 0755)
	dstPath := filepath.Join(uploadDir, filename)

	if err := c.SaveUploadedFile(file, dstPath); err != nil {
		httpx.RespondError(c, err)
		return
	}

	show, err := h.shows.UploadPoster(c.Request.Context(), organizerID, id, dstPath)
	if err != nil {
		httpx.RespondError(c, err)
		return
	}

	httpx.RespondOK(c, show)
}

// GetPoster — GET /shows/:id/poster
func (h *ShowHandler) GetPoster(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httpx.RespondError(c, model.ErrInvalid)
		return
	}

	posterPath, err := h.shows.GetPosterPath(c.Request.Context(), id)
	if err != nil {
		httpx.RespondError(c, err)
		return
	}

	c.File(posterPath)
}

// GenerateSeatMap — POST /shows/:id/seatmap
func (h *ShowHandler) GenerateSeatMap(c *gin.Context) {
	organizerID, ok := getUserID(c)
	if !ok {
		httpx.RespondError(c, model.ErrUnauthorized)
		return
	}

	showID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httpx.RespondError(c, model.ErrInvalid)
		return
	}

	var in model.GenerateSeatMapInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.RespondError(c, model.ErrInvalid)
		return
	}

	if err := h.shows.GenerateSeatMap(c.Request.Context(), organizerID, showID, in); err != nil {
		httpx.RespondError(c, err)
		return
	}

	httpx.RespondOK(c, gin.H{"message": "seatmap generated"})
}

// GetStats — GET /shows/:id/stats
func (h *ShowHandler) GetStats(c *gin.Context) {
	organizerID, ok := getUserID(c)
	if !ok {
		httpx.RespondError(c, model.ErrUnauthorized)
		return
	}

	showID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httpx.RespondError(c, model.ErrInvalid)
		return
	}

	stats, err := h.shows.GetStats(c.Request.Context(), organizerID, showID)
	if err != nil {
		httpx.RespondError(c, err)
		return
	}

	httpx.RespondOK(c, stats)
}

// Cancel — PUT /shows/:id/cancel
func (h *ShowHandler) Cancel(c *gin.Context) {
	organizerID, ok := getUserID(c)
	if !ok {
		httpx.RespondError(c, model.ErrUnauthorized)
		return
	}

	showID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httpx.RespondError(c, model.ErrInvalid)
		return
	}

	if err := h.shows.CancelShow(c.Request.Context(), organizerID, showID); err != nil {
		httpx.RespondError(c, err)
		return
	}

	httpx.RespondOK(c, gin.H{"message": "show cancelled and tickets refunded"})
}

