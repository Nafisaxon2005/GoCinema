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

func (s *ShowService) Create(ctx context.Context, organizerID int64, in model.CreateShowInput) (*model.Show, error) {
	if in.Title == "" || in.Venue == "" {
		return nil, model.ErrInvalid
	}
	if in.StartsAt.IsZero() || in.StartsAt.Before(time.Now()) {
		return nil, model.ErrInvalid
	}

	show := &model.Show{
		OrganizerID: organizerID,
		Title:       in.Title,
		Venue:       in.Venue,
		StartsAt:    in.StartsAt,
		Status:      model.ShowDraft,
	}

	id, err := s.shows.Create(ctx, show)
	if err != nil {
		return nil, err
	}
	show.ID = id
	return show, nil
}

func (s *ShowService) Update(ctx context.Context, organizerID int64, id int64, in model.UpdateShowInput) (*model.Show, error) {
	show, err := s.shows.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if show.OrganizerID != organizerID {
		return nil, model.ErrForbidden
	}

	if in.Status != nil {
		targetStatus := *in.Status
		switch show.Status {
		case model.ShowDraft:
			if targetStatus != model.ShowDraft && targetStatus != model.ShowPublished {
				return nil, model.ErrInvalid
			}
		case model.ShowPublished:
			if targetStatus != model.ShowPublished && targetStatus != model.ShowCancelled {
				return nil, model.ErrInvalid
			}
		case model.ShowCancelled:
			if targetStatus != model.ShowCancelled {
				return nil, model.ErrInvalid
			}
		default:
			return nil, model.ErrInvalid
		}
		show.Status = targetStatus
	}

	if in.Title != nil {
		if *in.Title == "" {
			return nil, model.ErrInvalid
		}
		show.Title = *in.Title
	}

	if in.Venue != nil {
		if *in.Venue == "" {
			return nil, model.ErrInvalid
		}
		show.Venue = *in.Venue
	}

	if in.StartsAt != nil {
		if in.StartsAt.Before(time.Now()) {
			return nil, model.ErrInvalid
		}
		show.StartsAt = *in.StartsAt
	}

	if err := s.shows.Update(ctx, show); err != nil {
		return nil, err
	}

	return show, nil
}

func (s *ShowService) Delete(ctx context.Context, organizerID int64, id int64) error {
	show, err := s.shows.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if show.OrganizerID != organizerID {
		return model.ErrForbidden
	}

	return s.shows.Delete(ctx, id)
}

func (s *ShowService) UploadPoster(ctx context.Context, organizerID int64, id int64, posterPath string) (*model.Show, error) {
	show, err := s.shows.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if show.OrganizerID != organizerID {
		return nil, model.ErrForbidden
	}

	if err := s.shows.UpdatePoster(ctx, id, posterPath); err != nil {
		return nil, err
	}

	show.PosterPath = posterPath
	return show, nil
}

func (s *ShowService) GetPosterPath(ctx context.Context, id int64) (string, error) {
	show, err := s.shows.GetByID(ctx, id)
	if err != nil {
		return "", err
	}

	if show.PosterPath == "" {
		return "", model.ErrNotFound
	}

	return show.PosterPath, nil
}

func (s *ShowService) GenerateSeatMap(ctx context.Context, organizerID int64, showID int64, in model.GenerateSeatMapInput) error {
	show, err := s.shows.GetByID(ctx, showID)
	if err != nil {
		return err
	}

	if show.OrganizerID != organizerID {
		return model.ErrForbidden
	}

	if in.Rows <= 0 || in.SeatsPerRow <= 0 || in.Price < 0 {
		return model.ErrInvalid
	}

	seats := make([]model.Seat, 0, in.Rows*in.SeatsPerRow)
	for r := 1; r <= in.Rows; r++ {
		price := in.Price
		for _, z := range in.Zones {
			if r >= z.FromRow && r <= z.ToRow {
				price = z.Price
				break
			}
		}

		for n := 1; n <= in.SeatsPerRow; n++ {
			seats = append(seats, model.Seat{
				ShowID: showID,
				Row:    r,
				Num:    n,
				Price:  price,
				Status: model.SeatFree,
			})
		}
	}

	return s.shows.GenerateSeatMap(ctx, showID, seats)
}

func (s *ShowService) GetStats(ctx context.Context, organizerID int64, showID int64) (*model.ShowStats, error) {
	show, err := s.shows.GetByID(ctx, showID)
	if err != nil {
		return nil, err
	}

	if show.OrganizerID != organizerID {
		return nil, model.ErrForbidden
	}

	return s.shows.GetStats(ctx, showID)
}

func (s *ShowService) CancelShow(ctx context.Context, organizerID int64, showID int64) error {
	show, err := s.shows.GetByID(ctx, showID)
	if err != nil {
		return err
	}

	if show.OrganizerID != organizerID {
		return model.ErrForbidden
	}

	return s.shows.CancelShow(ctx, showID)
}

