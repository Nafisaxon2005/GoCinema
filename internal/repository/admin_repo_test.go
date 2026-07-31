package repository

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/raxima/seatpicker/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestPgAdminRepo_GetStatsByPeriod(t *testing.T) {
	mock, err := pgxmock.NewConn()
	assert.NoError(t, err)
	defer func() { _ = mock.Close(context.Background()) }()

	r := NewPgAdminRepo(mock)

	from := time.Now().Add(-24 * time.Hour)
	to := time.Now()

	rows := pgxmock.NewRows([]string{"id", "title", "sold", "total", "revenue"}).
		AddRow(int64(1), "Movie 1", int64(5), int64(10), float64(500))

	mock.ExpectQuery(`SELECT s.id, s.title`).
		WithArgs(from, to).
		WillReturnRows(rows)

	stats, err := r.GetStatsByPeriod(context.Background(), from, to)
	assert.NoError(t, err)
	assert.Len(t, stats, 1)
	assert.Equal(t, "Movie 1", stats[0].Title)
}

func TestPgAdminRepo_ListShows(t *testing.T) {
	mock, err := pgxmock.NewConn()
	assert.NoError(t, err)
	defer func() { _ = mock.Close(context.Background()) }()

	r := NewPgAdminRepo(mock)
	status := model.ShowDraft

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM shows WHERE`).
		WithArgs(status).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))

	rows := pgxmock.NewRows([]string{"id", "organizer_id", "title", "venue", "starts_at", "status", "poster_path"}).
		AddRow(int64(1), int64(2), "Movie 1", "Hall A", time.Now(), status, "path")

	mock.ExpectQuery(`SELECT id, organizer_id, title, venue, starts_at, status`).
		WithArgs(status, 10, 0).
		WillReturnRows(rows)

	filter := AdminShowFilter{
		Status:   status,
		Page:     1,
		PageSize: 10,
	}

	shows, total, err := r.ListShows(context.Background(), filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, shows, 1)
}
