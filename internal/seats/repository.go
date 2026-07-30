package seats

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/raxima/seatpicker/internal/model"
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

// CancelBooking atomically deletes a booking and frees the associated seat.
func (r *Repository) CancelBooking(ctx context.Context, bookingID, userID int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	var ownerID int64
	var seatID int64

	// Fetch the booking details to verify ownership and get the seat ID
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

// GetShowSeats retrieves all seats for a specific show.
func (r *Repository) GetShowSeats(ctx context.Context, showID int64) ([]model.Seat, error) {
	query := `
		SELECT id, show_id, row, num, price, status 
		FROM seats 
		WHERE show_id = $1 
		ORDER BY row, num
	`

	rows, err := r.pool.Query(ctx, query, showID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var seats []model.Seat
	for rows.Next() {
		var s model.Seat
		if err := rows.Scan(&s.ID, &s.ShowID, &s.Row, &s.Num, &s.Price, &s.Status); err != nil {
			return nil, err
		}
		seats = append(seats, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return seats, nil
}
