package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/raxima/seatpicker/internal/repository"

	"github.com/raxima/seatpicker/internal/httpx"
	"github.com/raxima/seatpicker/internal/model"
	"github.com/raxima/seatpicker/internal/service"
)

const adminDateLayout = "2006-01-02"

type AdminHandler struct {
	service service.AdminService
}

func NewAdminHandler(s service.AdminService) *AdminHandler {
	return &AdminHandler{service: s}
}

func (h *AdminHandler) GetStats(c *gin.Context) {
	from, to, err := parsePeriod(c.Query("from"), c.Query("to"))
	if err != nil {
		httpx.RespondError(c, model.ErrInvalid)
		return
	}

	stats, err := h.service.GetStats(c.Request.Context(), from, to)
	if err != nil {
		httpx.RespondError(c, err)
		return
	}

	httpx.RespondOK(c, stats)
}

func parsePeriod(fromStr, toStr string) (time.Time, time.Time, error) {
	now := time.Now()

	to := now
	if toStr != "" {
		parsed, err := time.Parse(adminDateLayout, toStr)
		if err != nil {
			return time.Time{}, time.Time{}, model.ErrInvalid
		}
		to = parsed
	}

	from := to.AddDate(0, 0, -30)
	if fromStr != "" {
		parsed, err := time.Parse(adminDateLayout, fromStr)
		if err != nil {
			return time.Time{}, time.Time{}, model.ErrInvalid
		}
		from = parsed
	}

	return from, to, nil
}

func (h *AdminHandler) GetAllShows(c *gin.Context) {
	f := repository.AdminShowFilter{
		Search: c.Query("search"),
		Venue:  c.Query("venue"),
		Status: model.ShowStatus(c.Query("status")),
	}

	if orgIDStr := c.Query("organizer_id"); orgIDStr != "" {
		orgID, err := strconv.ParseInt(orgIDStr, 10, 64)
		if err != nil {
			httpx.RespondError(c, model.ErrInvalid)
			return
		}
		f.OrganizerID = orgID
	}

	if fromStr := c.Query("from"); fromStr != "" {
		parsed, err := time.Parse(adminDateLayout, fromStr)
		if err != nil {
			httpx.RespondError(c, model.ErrInvalid)
			return
		}
		f.DateFrom = &parsed
	}
	if toStr := c.Query("to"); toStr != "" {
		parsed, err := time.Parse(adminDateLayout, toStr)
		if err != nil {
			httpx.RespondError(c, model.ErrInvalid)
			return
		}
		f.DateTo = &parsed
	}

	f.Page, _ = strconv.Atoi(c.Query("page"))
	f.PageSize, _ = strconv.Atoi(c.Query("page_size"))

	shows, total, err := h.service.ListShows(c.Request.Context(), f)
	if err != nil {
		httpx.RespondError(c, err)
		return
	}

	httpx.RespondOK(c, model.ShowListResponse{
		Items:    shows,
		Total:    total,
		Page:     f.Page,
		PageSize: f.PageSize,
	})
}

type ModerateShowInput struct {
	Action string `json:"action" binding:"required,oneof=approve reject"`
}

func (h *AdminHandler) ModerateShow(c *gin.Context) {
	showID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httpx.RespondError(c, model.ErrInvalid)
		return
	}

	var input ModerateShowInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.RespondError(c, model.ErrInvalid)
		return
	}

	if err := h.service.ModerateShow(c.Request.Context(), showID, input.Action); err != nil {
		httpx.RespondError(c, err)
		return
	}

	httpx.RespondOK(c, gin.H{"status": "ok"})
}
