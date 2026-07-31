package router

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"

	"github.com/raxima/seatpicker/internal/service"
)

func TestNew(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("не удалось создать pgxmock: %v", err)
	}
	defer mock.Close()

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	authCfg := service.AuthConfig{
		JWTSecret:  []byte("test-secret"),
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 24 * time.Hour,
	}

	r := New(mock, logger, authCfg)
	if r == nil {
		t.Fatal("New вернул nil")
	}

	t.Run("маршрут /health зарегистрирован", func(t *testing.T) {
		mock.ExpectPing()

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("статус = %d, ожидалось %d, тело: %s", w.Code, http.StatusOK, w.Body.String())
		}
	})

	t.Run("маршрут /shows зарегистрирован", func(t *testing.T) {
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM shows`).
			WithArgs(pgxmock.AnyArg()).
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery(`SELECT id, organizer_id, title, venue, starts_at, status`).
			WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnRows(pgxmock.NewRows([]string{"id", "organizer_id", "title", "venue", "starts_at", "status", "poster_path"}))

		req := httptest.NewRequest(http.MethodGet, "/shows", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("статус = %d, ожидалось %d, тело: %s", w.Code, http.StatusOK, w.Body.String())
		}
	})

	t.Run("маршрут /auth/register зарегистрирован", func(t *testing.T) {
		mock.ExpectQuery(`INSERT INTO users`).
			WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnError(context.DeadlineExceeded)

		req := httptest.NewRequest(http.MethodPost, "/auth/register",
			strings.NewReader(`{"login":"x","password":"12345678","role":"viewer"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		
		if w.Code == http.StatusNotFound {
			t.Errorf("маршрут /auth/register не найден")
		}
	})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("не все ожидания выполнены: %v", err)
	}
}
