package seats

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/raxima/seatpicker/internal/model"
)

type Pool interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Repository struct {
	pool Pool
}

func NewRepository(pool Pool) *Repository {
	return &Repository{pool: pool}
}

// BookSeat атомарно переводит место free -> booked и создаёт бронь.
func (r *Repository) BookSeat(ctx context.Context, showID, seatID, userID int64) (bookingID int64, err error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}

	defer func() { _ = tx.Rollback(ctx) }()

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

	defer func() { _ = tx.Rollback(ctx) }()

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

// RefundBooking (admin-версия CancelBooking): помечает бронь как cancelled
// с указанием причины и освобождает место. В отличие от CancelBooking,
// не проверяет владельца — вызывается только из admin-хендлера.
func (r *Repository) RefundBooking(ctx context.Context, bookingID int64, reason string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var seatID int64
	err = tx.QueryRow(ctx,
		`SELECT seat_id FROM bookings WHERE id = $1 AND status = $2`,
		bookingID, model.BookingBooked,
	).Scan(&seatID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrBookingNotFound
		}
		return err
	}

	_, err = tx.Exec(ctx,
		`UPDATE bookings SET status = $1, reason = $2, refunded_at = now() WHERE id = $3`,
		model.BookingCancelled, reason, bookingID,
	)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(ctx,
		`UPDATE seats SET status = $1 WHERE id = $2`,
		model.SeatFree, seatID,
	); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// ListRefunds возвращает возвраты билетов с пагинацией (для админ-панели).
func (r *Repository) ListRefunds(ctx context.Context, limit, offset int) ([]model.RefundResponse, error) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, show_id, seat_id, user_id, reason, refunded_at
		 FROM bookings
		 WHERE status = $1
		 ORDER BY refunded_at DESC
		 LIMIT $2 OFFSET $3`,
		model.BookingCancelled, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.RefundResponse
	for rows.Next() {
		var v model.RefundResponse
		if err := rows.Scan(&v.BookingID, &v.ShowID, &v.SeatID, &v.UserID, &v.Reason, &v.RefundedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// ListStaleHolds возвращает места, которые держатся в статусе held
// дольше olderThan (для просмотра администратором, C-06).
func (r *Repository) ListStaleHolds(ctx context.Context, olderThan time.Duration) ([]model.HeldSeatResponse, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, show_id, held_at FROM seats
		 WHERE status = $1 AND held_at < now() - $2::interval
		 ORDER BY held_at`,
		model.SeatHeld, fmt.Sprintf("%d seconds", int(olderThan.Seconds())),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.HeldSeatResponse
	for rows.Next() {
		var v model.HeldSeatResponse
		if err := rows.Scan(&v.SeatID, &v.ShowID, &v.HeldAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// ReleaseStaleHolds атомарно переводит протухшие held-места обратно в free.
// Возвращает количество освобождённых мест (C-06, вызывается фоновой задачей).
func (r *Repository) ReleaseStaleHolds(ctx context.Context, olderThan time.Duration) (int64, error) {
	cmdTag, err := r.pool.Exec(ctx,
		`UPDATE seats SET status = $1, held_at = NULL
		 WHERE status = $2 AND held_at < now() - $3::interval`,
		model.SeatFree, model.SeatHeld, fmt.Sprintf("%d seconds", int(olderThan.Seconds())),
	)
	if err != nil {
		return 0, err
	}
	return cmdTag.RowsAffected(), nil
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

// GetUserBookings retrieves user bookings with optional filters (status, date) and pagination.
func (r *Repository) GetUserBookings(ctx context.Context, filter model.BookingFilter) ([]model.BookingResponse, error) {
	query := `
		SELECT b.id, b.show_id, b.seat_id, b.created_at, s.date, s.status 
		FROM bookings b
		JOIN shows s ON b.show_id = s.id
		WHERE b.user_id = $1
	`
	args := []interface{}{filter.UserID}
	argID := 2
	var conditions []string

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("s.status = $%d", argID))
		args = append(args, filter.Status)
		argID++
	}

	if filter.Date != "" {
		conditions = append(conditions, fmt.Sprintf("s.date::text LIKE $%d", argID))
		args = append(args, "%"+filter.Date+"%")
		argID++
	}

	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	// Дефолтные значения для пагинации, если не переданы
	limit := filter.Limit
	if limit <= 0 {
		limit = 10
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	query += fmt.Sprintf(" ORDER BY b.created_at DESC LIMIT $%d OFFSET $%d", argID, argID+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookings []model.BookingResponse
	for rows.Next() {
		var b model.BookingResponse
		var createdAt interface{}
		var showDate interface{}

		if err := rows.Scan(&b.ID, &b.ShowID, &b.SeatID, &createdAt, &showDate, &b.Status); err != nil {
			return nil, err
		}

		b.CreatedAt = fmt.Sprintf("%v", createdAt)
		b.ShowDate = fmt.Sprintf("%v", showDate)
		bookings = append(bookings, b)
	}

	return bookings, rows.Err()
}
