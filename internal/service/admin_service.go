package service

import (
	"context"
	"time"

	"github.com/raxima/seatpicker/internal/model"
	"github.com/raxima/seatpicker/internal/repository"
	"github.com/raxima/seatpicker/internal/seats"
)

type AdminService interface {
	GetStats(ctx context.Context, from, to time.Time) ([]model.ShowSalesStat, error)
	ListShows(ctx context.Context, f repository.AdminShowFilter) ([]model.Show, int, error)
	ModerateShow(ctx context.Context, showID int64, action string) error
	RefundBooking(ctx context.Context, bookingID int64, reason string) error
	ListRefunds(ctx context.Context, limit, offset int) ([]model.RefundResponse, error)
	ListStaleHolds(ctx context.Context, olderThan time.Duration) ([]model.HeldSeatResponse, error)
}

type adminService struct {
	repo      repository.AdminRepo
	showRepo  repository.ShowRepo
	seatsRepo *seats.Repository
}

func NewAdminService(repo repository.AdminRepo, showRepo repository.ShowRepo, seatsRepo *seats.Repository) AdminService {
	return &adminService{repo: repo, showRepo: showRepo, seatsRepo: seatsRepo}
}

func (s *adminService) GetStats(ctx context.Context, from, to time.Time) ([]model.ShowSalesStat, error) {
	if to.Before(from) {
		return nil, model.ErrInvalid
	}
	return s.repo.GetStatsByPeriod(ctx, from, to)
}

func (s *adminService) ListShows(ctx context.Context, f repository.AdminShowFilter) ([]model.Show, int, error) {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 || f.PageSize > 100 {
		f.PageSize = 20
	}
	return s.repo.ListShows(ctx, f)
}

func (s *adminService) ModerateShow(ctx context.Context, showID int64, action string) error {
	show, err := s.showRepo.GetByID(ctx, showID)
	if err != nil {
		return err
	}

	if show.Status != model.ShowDraft {
		return model.ErrInvalid
	}

	switch action {
	case "approve":
		return s.showRepo.UpdateStatus(ctx, showID, model.ShowPublished)
	case "reject":
		return s.showRepo.UpdateStatus(ctx, showID, model.ShowCancelled)
	default:
		return model.ErrInvalid
	}
}

// RefundBooking оформляет возврат билета с указанием причины.
func (s *adminService) RefundBooking(ctx context.Context, bookingID int64, reason string) error {
	if reason == "" {
		return model.ErrInvalid
	}
	return s.seatsRepo.RefundBooking(ctx, bookingID, reason)
}

// ListRefunds возвращает список оформленных возвратов с пагинацией.
func (s *adminService) ListRefunds(ctx context.Context, limit, offset int) ([]model.RefundResponse, error) {
	return s.seatsRepo.ListRefunds(ctx, limit, offset)
}

// ListStaleHolds возвращает места, застрявшие в статусе held дольше olderThan (C-06).
func (s *adminService) ListStaleHolds(ctx context.Context, olderThan time.Duration) ([]model.HeldSeatResponse, error) {
	return s.seatsRepo.ListStaleHolds(ctx, olderThan)
}
