package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/raxima/seatpicker/internal/model"
)

type ShowListFilter struct {
	Search   string // поиск по названию (ILIKE)
	Venue    string // точный фильтр по залу
	DateFrom *time.Time
	DateTo   *time.Time
	Status   model.ShowStatus
	Page     int
	PageSize int
}

type UserRepo interface {
	Create(ctx context.Context, u *model.User) (int64, error)
	GetByLogin(ctx context.Context, login string) (*model.User, error)
	GetByID(ctx context.Context, id int64) (*model.User, error)
}

type ShowRepo interface {
	Create(ctx context.Context, s *model.Show) (int64, error)
	GetByID(ctx context.Context, id int64) (*model.Show, error)
	List(ctx context.Context, f ShowListFilter) ([]model.Show, int, error)
	UpdateStatus(ctx context.Context, id int64, status model.ShowStatus) error
	Update(ctx context.Context, s *model.Show) error
	Delete(ctx context.Context, id int64) error
	UpdatePoster(ctx context.Context, showID int64, posterPath string) error
	GetStats(ctx context.Context, showID int64) (*model.ShowStats, error)
	GenerateSeatMap(ctx context.Context, showID int64, seats []model.Seat) error
	CancelShow(ctx context.Context, showID int64) error
	CountFreeSeats(ctx context.Context, showID int64) (int, error)
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

type RefreshTokenRepo interface {
	Create(ctx context.Context, rt *model.RefreshToken) (int64, error)
	GetByHash(ctx context.Context, hash string) (*model.RefreshToken, error)
	Revoke(ctx context.Context, id int64) error
	RevokeAllForUser(ctx context.Context, userID int64) error
}

type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}
