package handler

import (
	"strconv"
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
