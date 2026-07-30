package seats

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// BookSeat атомарно переводит место free -> booked и создаёт бронь.
func (r *Repository) BookSeat(ctx context.Context, showID, seatID, userID int64) (bookingID int64, err error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}

	defer tx.Rollback(ctx)

	var showStatus string
	err = tx.QueryRow(ctx, `SELECT status FROM shows WHERE id = $1`, showID).Scan(&showStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrShowNotAvailable
		}
		return 0, err
	}
	if showStatus != "published" {
		return 0, ErrShowNotAvailable
	}

	cmdTag, err := tx.Exec(ctx,
		`UPDATE seats
		 SET status = 'booked'
		 WHERE id = $1 AND show_id = $2 AND status = 'free'`,
		seatID, showID,
	)
	if err != nil {
		return 0, err
	}

	if cmdTag.RowsAffected() == 0 {
		// Либо место не найдено либо  его только что забрал кто-то другой
		return 0, ErrSeatTaken
	}

	err = tx.QueryRow(ctx,
		`INSERT INTO bookings (show_id, seat_id, user_id, created_at)
		 VALUES ($1, $2, $3, now())
		 RETURNING id`,
		showID, seatID, userID,
	).Scan(&bookingID)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return bookingID, nil
}

func (r *Repository) CancelBooking(ctx context.Context, bookingID, userID int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	var ownerID int64
	var seatID int64

	// Получает данные о бронировании, чтобы проверить владельца и получить ID места.
	err = tx.QueryRow(ctx, `SELECT user_id, seat_id FROM bookings WHERE id = $1`, bookingID).Scan(&ownerID, &seatID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrBookingNotFound
		}
		return err
	}
	// Verify that the user cancelling the booking is the actual owner
	if ownerID != userID {
		return ErrForbidden
	}

	// Delete the booking record
	_, err = tx.Exec(ctx, `DELETE FROM bookings WHERE id = $1`, bookingID)
	if err != nil {
		return err
	}

	// Free the seat
	_, err = tx.Exec(ctx, `UPDATE seats SET status = 'free' WHERE id = $1`, seatID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
