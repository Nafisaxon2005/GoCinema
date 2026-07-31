package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/raxima/seatpicker/internal/model"
)

type AdminRepo interface {
	GetStatsByPeriod(ctx context.Context, from, to time.Time) ([]model.ShowSalesStat, error)
	ListShows(ctx context.Context, f AdminShowFilter) ([]model.Show, int, error)
}

type pgAdminRepo struct {
	db DBTX
}

func NewPgAdminRepo(db DBTX) AdminRepo {
	return &pgAdminRepo{db: db}
}

func (r *pgAdminRepo) GetStatsByPeriod(ctx context.Context, from, to time.Time) ([]model.ShowSalesStat, error) {
	const query = `
		SELECT s.id, s.title,
		       COUNT(*) FILTER (WHERE se.status = 'booked') AS sold,
		       COUNT(*) AS total,
		       COALESCE(SUM(se.price) FILTER (WHERE se.status = 'booked'), 0) AS revenue
		FROM shows s
		JOIN seats se ON se.show_id = s.id
		WHERE s.starts_at BETWEEN $1 AND $2
		GROUP BY s.id, s.title
		ORDER BY s.starts_at
	`

	rows, err := r.db.Query(ctx, query, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make([]model.ShowSalesStat, 0)
	for rows.Next() {
		var st model.ShowSalesStat
		if err := rows.Scan(&st.ShowID, &st.Title, &st.Sold, &st.Total, &st.Revenue); err != nil {
			return nil, err
		}
		stats = append(stats, st)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return stats, nil
}

type AdminShowFilter struct {
	Search      string
	Venue       string
	OrganizerID int64
	Status      model.ShowStatus // пусто = любой статус
	DateFrom    *time.Time
	DateTo      *time.Time
	Page        int
	PageSize    int
}

// ListShows возвращает сеансы с любым статусом (в отличие от публичной
// афиши A-01, которая жёстко фильтрует только published). Платформа (C-02)
// должна видеть draft/published/cancelled для модерации и аналитики.
func (r *pgAdminRepo) ListShows(ctx context.Context, f AdminShowFilter) ([]model.Show, int, error) {
	var (
		conditions []string
		args       []any
	)
	idx := 1

	if f.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", idx))
		args = append(args, f.Status)
		idx++
	}
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
	if f.OrganizerID != 0 {
		conditions = append(conditions, fmt.Sprintf("organizer_id = $%d", idx))
		args = append(args, f.OrganizerID)
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

	where := "TRUE"
	if len(conditions) > 0 {
		where = strings.Join(conditions, " AND ")
	}

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

	shows := make([]model.Show, 0)
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
