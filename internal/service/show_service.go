package service

import (
	"context"
	"time"

	"github.com/raxima/seatpicker/internal/model"
	"github.com/raxima/seatpicker/internal/repository"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

type ShowService struct {
	shows repository.ShowRepo
}

func NewShowService(shows repository.ShowRepo) *ShowService {
	return &ShowService{shows: shows}
}

type ListShowsInput struct {
	Search   string
	Venue    string
	DateFrom *time.Time
	DateTo   *time.Time
	Page     int
	PageSize int
}

// List отдаёт только published-сеансы — зритель не может запросить draft/cancelled
func (s *ShowService) List(ctx context.Context, in ListShowsInput) (*model.ShowListResponse, error) {
	page := in.Page
	if page < 1 {
		page = 1
	}
	pageSize := in.PageSize
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	items, total, err := s.shows.List(ctx, repository.ShowListFilter{
		Search:   in.Search,
		Venue:    in.Venue,
		DateFrom: in.DateFrom,
		DateTo:   in.DateTo,
		Status:   model.ShowPublished,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, err
	}

	return &model.ShowListResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// GetByID отдаёт деталь только для published-сеансов — по прямой ссылке
// на draft/cancelled зритель получает 404, как будто сеанса не существует.
func (s *ShowService) GetByID(ctx context.Context, id int64) (*model.ShowDetail, error) {
	show, err := s.shows.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if show.Status != model.ShowPublished {
		return nil, model.ErrNotFound
	}

	free, err := s.shows.CountFreeSeats(ctx, id)
	if err != nil {
		return nil, err
	}

	return &model.ShowDetail{Show: *show, FreeSeats: free}, nil
}
