package repository

import (
	"context"

	"github.com/raxima/seatpicker/internal/model"
)

type UserRepo interface {
	Create(ctx context.Context, u *model.User) (int64, error)
	GetByLogin(ctx context.Context, login string) (*model.User, error)
	GetByID(ctx context.Context, id int64) (*model.User, error)
}

type ShowRepo interface {
	Create(ctx context.Context, s *model.Show) (int64, error)
	GetByID(ctx context.Context, id int64) (*model.Show, error)
	List(ctx context.Context) ([]model.Show, error)
	UpdateStatus(ctx context.Context, id int64, status model.ShowStatus) error
}

type SeatRepo interface {
	BulkCreate(ctx context.Context, seats []model.Seat) error
	ListByShow(ctx context.Context, showID int64) ([]model.Seat, error)
	TryBook(ctx context.Context, seatID int64) error
}

type BookingRepo interface {
	Create(ctx context.Context, b *model.Booking) (int64, error)
	ListByUser(ctx context.Context, userID int64) ([]model.Booking, error)
	ListByShow(ctx context.Context, showID int64) ([]model.Booking, error)
	Cancel(ctx context.Context, id int64) error
}
