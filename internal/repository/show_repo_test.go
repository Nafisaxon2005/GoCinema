package repository

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"

	"github.com/raxima/seatpicker/internal/model"
)

func TestPgShowRepo_Create(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("не удалось создать pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewPgShowRepo(mock)
	startsAt := time.Now().Add(24 * time.Hour)

	mock.ExpectQuery(`INSERT INTO shows`).
		WithArgs(int64(1), "Дюна", "Зал 1", startsAt, model.ShowDraft, "").
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(100)))

	id, err := repo.Create(context.Background(), &model.Show{
		OrganizerID: 1, Title: "Дюна", Venue: "Зал 1", StartsAt: startsAt, Status: model.ShowDraft,
	})
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if id != 100 {
		t.Errorf("id = %d, ожидалось 100", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("не все ожидания выполнены: %v", err)
	}
}

func TestPgShowRepo_GetByID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("не удалось создать pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewPgShowRepo(mock)
	now := time.Now()

	mock.ExpectQuery(`SELECT id, organizer_id, title, venue, starts_at, status`).
		WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows([]string{"id", "organizer_id", "title", "venue", "starts_at", "status", "poster_path"}).
			AddRow(int64(1), int64(2), "Дюна", "Зал 1", now, model.ShowPublished, "poster.jpg"))

	s, err := repo.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if s.Title != "Дюна" || s.Status != model.ShowPublished {
		t.Errorf("неверные данные сеанса: %+v", s)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("не все ожидания выполнены: %v", err)
	}
}

func TestPgShowRepo_UpdateStatus(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("не удалось создать pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewPgShowRepo(mock)

	t.Run("успешное обновление", func(t *testing.T) {
		mock.ExpectExec(`UPDATE shows SET status`).
			WithArgs(int64(1), model.ShowPublished).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		if err := repo.UpdateStatus(context.Background(), 1, model.ShowPublished); err != nil {
			t.Fatalf("неожиданная ошибка: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("не все ожидания выполнены: %v", err)
		}
	})

	t.Run("сеанс не найден -> ErrNotFound", func(t *testing.T) {
		mock.ExpectExec(`UPDATE shows SET status`).
			WithArgs(int64(999), model.ShowCancelled).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		err := repo.UpdateStatus(context.Background(), 999, model.ShowCancelled)
		if err != model.ErrNotFound {
			t.Errorf("ожидалась ошибка %v, получено %v", model.ErrNotFound, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("не все ожидания выполнены: %v", err)
		}
	})
}

func TestPgShowRepo_List(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("не удалось создать pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewPgShowRepo(mock)
	now := time.Now()

	t.Run("список без доп. фильтров", func(t *testing.T) {
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM shows`).
			WithArgs(model.ShowPublished).
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(2))

		mock.ExpectQuery(`SELECT id, organizer_id, title, venue, starts_at, status`).
			WithArgs(model.ShowPublished, 20, 0).
			WillReturnRows(pgxmock.NewRows([]string{"id", "organizer_id", "title", "venue", "starts_at", "status", "poster_path"}).
				AddRow(int64(1), int64(1), "Дюна", "Зал 1", now, model.ShowPublished, "").
				AddRow(int64(2), int64(1), "Оппенгеймер", "Зал 2", now, model.ShowPublished, ""))

		shows, total, err := repo.List(context.Background(), ShowListFilter{
			Status: model.ShowPublished, Page: 1, PageSize: 20,
		})
		if err != nil {
			t.Fatalf("неожиданная ошибка: %v", err)
		}
		if total != 2 || len(shows) != 2 {
			t.Errorf("total = %d, len(shows) = %d, ожидалось 2 и 2", total, len(shows))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("не все ожидания выполнены: %v", err)
		}
	})

	t.Run("с фильтром по поиску и залу", func(t *testing.T) {
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM shows`).
			WithArgs(model.ShowPublished, "%Дюна%", "Зал 1").
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))

		mock.ExpectQuery(`SELECT id, organizer_id, title, venue, starts_at, status`).
			WithArgs(model.ShowPublished, "%Дюна%", "Зал 1", 20, 0).
			WillReturnRows(pgxmock.NewRows([]string{"id", "organizer_id", "title", "venue", "starts_at", "status", "poster_path"}).
				AddRow(int64(1), int64(1), "Дюна", "Зал 1", now, model.ShowPublished, ""))

		shows, total, err := repo.List(context.Background(), ShowListFilter{
			Status: model.ShowPublished, Search: "Дюна", Venue: "Зал 1", Page: 1, PageSize: 20,
		})
		if err != nil {
			t.Fatalf("неожиданная ошибка: %v", err)
		}
		if total != 1 || len(shows) != 1 {
			t.Errorf("total = %d, len(shows) = %d, ожидалось 1 и 1", total, len(shows))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("не все ожидания выполнены: %v", err)
		}
	})

	t.Run("ошибка при подсчёте total пробрасывается", func(t *testing.T) {
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM shows`).
			WithArgs(model.ShowDraft).
			WillReturnError(context.DeadlineExceeded)

		_, _, err := repo.List(context.Background(), ShowListFilter{
			Status: model.ShowDraft, Page: 1, PageSize: 20,
		})
		if err == nil {
			t.Fatal("ожидалась ошибка")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("не все ожидания выполнены: %v", err)
		}
	})
}

func TestPgShowRepo_CountFreeSeats(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("не удалось создать pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewPgShowRepo(mock)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM seats`).
		WithArgs(int64(1), model.SeatFree).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(42))

	count, err := repo.CountFreeSeats(context.Background(), 1)
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if count != 42 {
		t.Errorf("count = %d, ожидалось 42", count)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("не все ожидания выполнены: %v", err)
	}
}
