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

func TestPgShowRepo_GetStats(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("не удалось создать pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewPgShowRepo(mock)

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM shows WHERE id = \$1\)`).
		WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectQuery(`SELECT\s+COUNT\(\*\) AS total_seats`).
		WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows([]string{"total_seats", "sold_seats", "revenue"}).
			AddRow(100, 25, int64(12500)))

	stats, err := repo.GetStats(context.Background(), 1)
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if stats.TotalSeats != 100 || stats.SoldSeats != 25 || stats.Revenue != 12500 || stats.OccupancyRate != 25.0 {
		t.Errorf("некорректная статистика: %+v", stats)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("не все ожидания выполнены: %v", err)
	}
}

func TestPgShowRepo_CancelShow(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("не удалось создать pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewPgShowRepo(mock)

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE shows SET status`).
		WithArgs(int64(1), model.ShowCancelled).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectExec(`UPDATE bookings SET status`).
		WithArgs(int64(1), model.BookingCancelled, model.BookingBooked).
		WillReturnResult(pgxmock.NewResult("UPDATE", 5))
	mock.ExpectExec(`UPDATE seats SET status`).
		WithArgs(int64(1), model.SeatFree).
		WillReturnResult(pgxmock.NewResult("UPDATE", 10))
	mock.ExpectCommit()

	if err := repo.CancelShow(context.Background(), 1); err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("не все ожидания выполнены: %v", err)
	}
}

func TestPgShowRepo_Update_Delete_Poster_SeatMap(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("не удалось создать pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewPgShowRepo(mock)
	now := time.Now()

	t.Run("Update", func(t *testing.T) {
		mock.ExpectExec(`UPDATE shows SET title`).
			WithArgs(int64(1), "Дюна 2", "Зал 1", now, model.ShowPublished).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := repo.Update(context.Background(), &model.Show{
			ID: 1, Title: "Дюна 2", Venue: "Зал 1", StartsAt: now, Status: model.ShowPublished,
		})
		if err != nil {
			t.Fatalf("неожиданная ошибка: %v", err)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		mock.ExpectExec(`DELETE FROM shows WHERE id = \$1`).
			WithArgs(int64(1)).
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		err := repo.Delete(context.Background(), 1)
		if err != nil {
			t.Fatalf("неожиданная ошибка: %v", err)
		}
	})

	t.Run("UpdatePoster", func(t *testing.T) {
		mock.ExpectExec(`UPDATE shows SET poster_path`).
			WithArgs(int64(1), "new_poster.jpg").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := repo.UpdatePoster(context.Background(), 1, "new_poster.jpg")
		if err != nil {
			t.Fatalf("неожиданная ошибка: %v", err)
		}
	})

	t.Run("GenerateSeatMap", func(t *testing.T) {
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM seats WHERE show_id = \$1 AND status = 'booked'`).
			WithArgs(int64(1)).
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

		mock.ExpectExec(`DELETE FROM seats WHERE show_id = \$1`).
			WithArgs(int64(1)).
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		mock.ExpectExec(`INSERT INTO seats`).
			WithArgs(int64(1), 1, 1, int64(500), model.SeatFree, int64(1), 1, 2, int64(500), model.SeatFree).
			WillReturnResult(pgxmock.NewResult("INSERT", 2))

		seats := []model.Seat{
			{Row: 1, Num: 1, Price: 500},
			{Row: 1, Num: 2, Price: 500},
		}
		err := repo.GenerateSeatMap(context.Background(), 1, seats)
		if err != nil {
			t.Fatalf("неожиданная ошибка: %v", err)
		}
	})
}


