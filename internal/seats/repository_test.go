package seats

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/raxima/seatpicker/internal/model"
)

func TestRepository_BookSeat(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewRepository(mock)

	t.Run("successful booking", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT status FROM shows WHERE id = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(pgxmock.NewRows([]string{"status"}).AddRow("published"))

		mock.ExpectExec(`UPDATE seats SET status = 'booked'`).
			WithArgs(int64(10), int64(1)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		mock.ExpectQuery(`INSERT INTO bookings`).
			WithArgs(int64(1), int64(10), int64(100)).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(500)))

		mock.ExpectCommit()

		bookingID, err := repo.BookSeat(context.Background(), 1, 10, 100)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if bookingID != 500 {
			t.Errorf("bookingID = %d, expected 500", bookingID)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unfulfilled expectations: %v", err)
		}
	})

	t.Run("show not published -> ErrShowNotAvailable", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT status FROM shows WHERE id = \$1`).
			WithArgs(int64(1)).
			WillReturnRows(pgxmock.NewRows([]string{"status"}).AddRow("draft"))

		bookingID, err := repo.BookSeat(context.Background(), 1, 10, 100)
		if err != ErrShowNotAvailable {
			t.Errorf("expected ErrShowNotAvailable, got %v", err)
		}
		if bookingID != 0 {
			t.Errorf("expected bookingID = 0, got %d", bookingID)
		}
	})
}

func TestRepository_CancelBooking(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewRepository(mock)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT user_id, seat_id FROM bookings WHERE id = \$1`).
		WithArgs(int64(500)).
		WillReturnRows(pgxmock.NewRows([]string{"user_id", "seat_id"}).AddRow(int64(100), int64(10)))

	mock.ExpectExec(`DELETE FROM bookings WHERE id = \$1`).
		WithArgs(int64(500)).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	mock.ExpectExec(`UPDATE seats SET status = 'free' WHERE id = \$1`).
		WithArgs(int64(10)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	mock.ExpectCommit()

	err = repo.CancelBooking(context.Background(), 500, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestRepository_GetShowSeats(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewRepository(mock)

	mock.ExpectQuery(`SELECT id, show_id, row, num, price, status FROM seats WHERE show_id = \$1`).
		WithArgs(int64(1)).
		WillReturnRows(pgxmock.NewRows([]string{"id", "show_id", "row", "num", "price", "status"}).
			AddRow(int64(10), int64(1), 1, 1, int64(500), model.SeatFree))

	seats, err := repo.GetShowSeats(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(seats) != 1 || seats[0].ID != 10 {
		t.Errorf("incorrect seats returned: %+v", seats)
	}
}

func TestRepository_GetUserBookings(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewRepository(mock)

	now := time.Now()
	mock.ExpectQuery(`SELECT b.id, b.show_id, b.seat_id, b.created_at, s.date, s.status FROM bookings b`).
		WithArgs(int64(100), 10, 0).
		WillReturnRows(pgxmock.NewRows([]string{"id", "show_id", "seat_id", "created_at", "date", "status"}).
			AddRow(int64(500), int64(1), int64(10), now, "2026-08-01", "published"))

	filter := model.BookingFilter{UserID: 100, Limit: 10, Offset: 0}
	bookings, err := repo.GetUserBookings(context.Background(), filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bookings) != 1 || bookings[0].ID != 500 {
		t.Errorf("incorrect bookings returned: %+v", bookings)
	}
}
