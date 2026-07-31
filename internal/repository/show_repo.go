package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/raxima/seatpicker/internal/model"
)

type PgShowRepo struct {
	db DBTX
}

func NewPgShowRepo(db DBTX) *PgShowRepo {
	return &PgShowRepo{db: db}
}

func (r *PgShowRepo) Create(ctx context.Context, s *model.Show) (int64, error) {
	const q = `
		INSERT INTO shows (organizer_id, title, venue, starts_at, status, poster_path)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`

	var id int64
	err := r.db.QueryRow(ctx, q, s.OrganizerID, s.Title, s.Venue, s.StartsAt, s.Status, s.PosterPath).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *PgShowRepo) GetByID(ctx context.Context, id int64) (*model.Show, error) {
	const q = `
		SELECT id, organizer_id, title, venue, starts_at, status, COALESCE(poster_path, '')
		FROM shows
		WHERE id = $1`

	var s model.Show
	err := r.db.QueryRow(ctx, q, id).Scan(
		&s.ID, &s.OrganizerID, &s.Title, &s.Venue, &s.StartsAt, &s.Status, &s.PosterPath,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *PgShowRepo) UpdateStatus(ctx context.Context, id int64, status model.ShowStatus) error {
	const q = `UPDATE shows SET status = $2 WHERE id = $1`

	tag, err := r.db.Exec(ctx, q, id, status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}

// total (для пагинации) и сам список с LIMIT/OFFSET.
func (r *PgShowRepo) List(ctx context.Context, f ShowListFilter) ([]model.Show, int, error) {
	var (
		conditions []string
		args       []any
	)
	idx := 1

	conditions = append(conditions, fmt.Sprintf("status = $%d", idx))
	args = append(args, f.Status)
	idx++

	if f.Search != "" {
		conditions = append(conditions, fmt.Sprintf("title ILIKE $%d", idx))
		args = append(args, "%"+f.Search+"%")
		idx++
	}
	if f.Venue != "" {
		conditions = append(conditions, fmt.Sprintf("venue = $%d", idx))
		args = append(args, f.Venue)
		idx++
	}
	if f.DateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("starts_at >= $%d", idx))
		args = append(args, *f.DateFrom)
		idx++
	}
	if f.DateTo != nil {
		conditions = append(conditions, fmt.Sprintf("starts_at < $%d", idx))
		args = append(args, *f.DateTo)
		idx++
	}

	where := strings.Join(conditions, " AND ")

	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM shows WHERE %s`, where)
	var total int
	if err := r.db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limitIdx, offsetIdx := idx, idx+1
	listQ := fmt.Sprintf(`
		SELECT id, organizer_id, title, venue, starts_at, status, COALESCE(poster_path, '')
		FROM shows
		WHERE %s
		ORDER BY starts_at ASC
		LIMIT $%d OFFSET $%d`, where, limitIdx, offsetIdx)

	listArgs := append(append([]any{}, args...), f.PageSize, (f.Page-1)*f.PageSize)

	rows, err := r.db.Query(ctx, listQ, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var shows []model.Show
	for rows.Next() {
		var s model.Show
		if err := rows.Scan(&s.ID, &s.OrganizerID, &s.Title, &s.Venue, &s.StartsAt, &s.Status, &s.PosterPath); err != nil {
			return nil, 0, err
		}
		shows = append(shows, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return shows, total, nil
}

func (r *PgShowRepo) CountFreeSeats(ctx context.Context, showID int64) (int, error) {
	const q = `SELECT COUNT(*) FROM seats WHERE show_id = $1 AND status = $2`

	var count int
	err := r.db.QueryRow(ctx, q, showID, model.SeatFree).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *PgShowRepo) Update(ctx context.Context, s *model.Show) error {
	const q = `UPDATE shows SET title = $2, venue = $3, starts_at = $4, status = $5 WHERE id = $1`

	tag, err := r.db.Exec(ctx, q, s.ID, s.Title, s.Venue, s.StartsAt, s.Status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}

func (r *PgShowRepo) Delete(ctx context.Context, id int64) error {
	const q = `DELETE FROM shows WHERE id = $1`

	tag, err := r.db.Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}

func (r *PgShowRepo) UpdatePoster(ctx context.Context, showID int64, posterPath string) error {
	const q = `UPDATE shows SET poster_path = $2 WHERE id = $1`

	tag, err := r.db.Exec(ctx, q, showID, posterPath)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}

func (r *PgShowRepo) GetStats(ctx context.Context, showID int64) (*model.ShowStats, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM shows WHERE id = $1)`, showID).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, model.ErrNotFound
	}

	const q = `
		SELECT
			COUNT(*) AS total_seats,
			COUNT(CASE WHEN status = 'booked' THEN 1 END) AS sold_seats,
			COALESCE(SUM(CASE WHEN status = 'booked' THEN price ELSE 0 END), 0) AS revenue
		FROM seats
		WHERE show_id = $1`

	var stats model.ShowStats
	stats.ShowID = showID
	err = r.db.QueryRow(ctx, q, showID).Scan(&stats.TotalSeats, &stats.SoldSeats, &stats.Revenue)
	if err != nil {
		return nil, err
	}

	if stats.TotalSeats > 0 {
		stats.OccupancyRate = (float64(stats.SoldSeats) / float64(stats.TotalSeats)) * 100.0
	} else {
		stats.OccupancyRate = 0.0
	}

	return &stats, nil
}

func (r *PgShowRepo) GenerateSeatMap(ctx context.Context, showID int64, seats []model.Seat) error {
	var bookedCount int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM seats WHERE show_id = $1 AND status = 'booked'`, showID).Scan(&bookedCount)
	if err != nil {
		return err
	}
	if bookedCount > 0 {
		return model.ErrSeatTaken
	}

	_, err = r.db.Exec(ctx, `DELETE FROM seats WHERE show_id = $1`, showID)
	if err != nil {
		return err
	}

	if len(seats) == 0 {
		return nil
	}

	valueStrings := make([]string, 0, len(seats))
	valueArgs := make([]any, 0, len(seats)*5)

	for i, seat := range seats {
		n := i * 5
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d)", n+1, n+2, n+3, n+4, n+5))
		valueArgs = append(valueArgs, showID, seat.Row, seat.Num, seat.Price, model.SeatFree)
	}

	stmt := fmt.Sprintf("INSERT INTO seats (show_id, row, num, price, status) VALUES %s", strings.Join(valueStrings, ", "))
	_, err = r.db.Exec(ctx, stmt, valueArgs...)
	return err
}

type Beginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

func (r *PgShowRepo) CancelShow(ctx context.Context, showID int64) error {
	beginner, ok := r.db.(Beginner)
	if !ok {
		tag, err := r.db.Exec(ctx, `UPDATE shows SET status = $2 WHERE id = $1`, showID, model.ShowCancelled)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return model.ErrNotFound
		}
		_, _ = r.db.Exec(ctx, `UPDATE bookings SET status = $2 WHERE show_id = $1 AND status = $3`, showID, model.BookingCancelled, model.BookingBooked)
		_, _ = r.db.Exec(ctx, `UPDATE seats SET status = $2 WHERE show_id = $1`, showID, model.SeatFree)
		return nil
	}

	tx, err := beginner.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `UPDATE shows SET status = $2 WHERE id = $1`, showID, model.ShowCancelled)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}

	_, err = tx.Exec(ctx, `UPDATE bookings SET status = $2 WHERE show_id = $1 AND status = $3`, showID, model.BookingCancelled, model.BookingBooked)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `UPDATE seats SET status = $2 WHERE show_id = $1`, showID, model.SeatFree)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

