package seats

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/raxima/seatpicker/internal/model"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type mockBooker struct {
	bookSeatFn        func(ctx context.Context, showID, seatID, userID int64) (int64, error)
	cancelBookingFn   func(ctx context.Context, bookingID, userID int64) error
	getShowSeatsFn    func(ctx context.Context, showID int64) ([]model.Seat, error)
	getUserBookingsFn func(ctx context.Context, filter model.BookingFilter) ([]model.BookingResponse, error)
}

func (m *mockBooker) BookSeat(ctx context.Context, showID, seatID, userID int64) (int64, error) {
	if m.bookSeatFn != nil {
		return m.bookSeatFn(ctx, showID, seatID, userID)
	}
	return 1, nil
}

func (m *mockBooker) CancelBooking(ctx context.Context, bookingID, userID int64) error {
	if m.cancelBookingFn != nil {
		return m.cancelBookingFn(ctx, bookingID, userID)
	}
	return nil
}

func (m *mockBooker) GetShowSeats(ctx context.Context, showID int64) ([]model.Seat, error) {
	if m.getShowSeatsFn != nil {
		return m.getShowSeatsFn(ctx, showID)
	}
	return nil, nil
}

func (m *mockBooker) GetUserBookings(ctx context.Context, filter model.BookingFilter) ([]model.BookingResponse, error) {
	if m.getUserBookingsFn != nil {
		return m.getUserBookingsFn(ctx, filter)
	}
	return nil, nil
}

func performTestRequest(handlerFunc gin.HandlerFunc, method, url string, userID *int64, params gin.Params) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest(method, url, nil)
	c.Request = req
	c.Params = params

	if userID != nil {
		c.Set("userID", *userID)
	}

	handlerFunc(c)
	return w
}

func TestHandler_Book(t *testing.T) {
	uid := int64(1)

	tests := []struct {
		name       string
		url        string
		params     gin.Params
		userID     *int64
		bookFn     func(ctx context.Context, showID, seatID, userID int64) (int64, error)
		wantStatus int
	}{
		{
			name:       "success booking",
			url:        "/shows/1/seats/2/book",
			params:     gin.Params{{Key: "id", Value: "1"}, {Key: "seatId", Value: "2"}},
			userID:     &uid,
			bookFn:     func(ctx context.Context, showID, seatID, userID int64) (int64, error) { return 10, nil },
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalid show id",
			url:        "/shows/abc/seats/2/book",
			params:     gin.Params{{Key: "id", Value: "abc"}, {Key: "seatId", Value: "2"}},
			userID:     &uid,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid seat id",
			url:        "/shows/1/seats/abc/book",
			params:     gin.Params{{Key: "id", Value: "1"}, {Key: "seatId", Value: "abc"}},
			userID:     &uid,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unauthorized",
			url:        "/shows/1/seats/2/book",
			params:     gin.Params{{Key: "id", Value: "1"}, {Key: "seatId", Value: "2"}},
			userID:     nil,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "seat taken conflict",
			url:        "/shows/1/seats/2/book",
			params:     gin.Params{{Key: "id", Value: "1"}, {Key: "seatId", Value: "2"}},
			userID:     &uid,
			bookFn:     func(ctx context.Context, showID, seatID, userID int64) (int64, error) { return 0, ErrSeatTaken },
			wantStatus: http.StatusConflict,
		},
		{
			name:       "show not available bad request",
			url:        "/shows/1/seats/2/book",
			params:     gin.Params{{Key: "id", Value: "1"}, {Key: "seatId", Value: "2"}},
			userID:     &uid,
			bookFn:     func(ctx context.Context, showID, seatID, userID int64) (int64, error) { return 0, ErrShowNotAvailable },
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "internal error",
			url:    "/shows/1/seats/2/book",
			params: gin.Params{{Key: "id", Value: "1"}, {Key: "seatId", Value: "2"}},
			userID: &uid,
			bookFn: func(ctx context.Context, showID, seatID, userID int64) (int64, error) {
				return 0, errors.New("db error")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockBooker{bookSeatFn: tt.bookFn}
			h := NewHandler(repo)

			w := performTestRequest(h.Book, http.MethodPost, tt.url, tt.userID, tt.params)
			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestHandler_GetSeats(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		params     gin.Params
		getSeatsFn func(ctx context.Context, showID int64) ([]model.Seat, error)
		wantStatus int
	}{
		{
			name:       "success get seats",
			url:        "/shows/1/seats",
			params:     gin.Params{{Key: "id", Value: "1"}},
			getSeatsFn: func(ctx context.Context, showID int64) ([]model.Seat, error) { return []model.Seat{{ID: 1}}, nil },
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid show id",
			url:        "/shows/abc/seats",
			params:     gin.Params{{Key: "id", Value: "abc"}},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "repo error internal",
			url:        "/shows/1/seats",
			params:     gin.Params{{Key: "id", Value: "1"}},
			getSeatsFn: func(ctx context.Context, showID int64) ([]model.Seat, error) { return nil, errors.New("error") },
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockBooker{getShowSeatsFn: tt.getSeatsFn}
			h := NewHandler(repo)

			w := performTestRequest(h.GetSeats, http.MethodGet, tt.url, nil, tt.params)
			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestHandler_Cancel(t *testing.T) {
	uid := int64(1)

	tests := []struct {
		name       string
		url        string
		params     gin.Params
		userID     *int64
		cancelFn   func(ctx context.Context, bookingID, userID int64) error
		wantStatus int
	}{
		{
			name:       "success cancel",
			url:        "/bookings/1",
			params:     gin.Params{{Key: "bookingId", Value: "1"}},
			userID:     &uid,
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "invalid booking id",
			url:        "/bookings/abc",
			params:     gin.Params{{Key: "bookingId", Value: "abc"}},
			userID:     &uid,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unauthorized",
			url:        "/bookings/1",
			params:     gin.Params{{Key: "bookingId", Value: "1"}},
			userID:     nil,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "booking not found",
			url:        "/bookings/1",
			params:     gin.Params{{Key: "bookingId", Value: "1"}},
			userID:     &uid,
			cancelFn:   func(ctx context.Context, bookingID, userID int64) error { return ErrBookingNotFound },
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "forbidden error",
			url:        "/bookings/1",
			params:     gin.Params{{Key: "bookingId", Value: "1"}},
			userID:     &uid,
			cancelFn:   func(ctx context.Context, bookingID, userID int64) error { return ErrForbidden },
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "internal error",
			url:        "/bookings/1",
			params:     gin.Params{{Key: "bookingId", Value: "1"}},
			userID:     &uid,
			cancelFn:   func(ctx context.Context, bookingID, userID int64) error { return errors.New("err") },
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockBooker{cancelBookingFn: tt.cancelFn}
			h := NewHandler(repo)

			w := performTestRequest(h.Cancel, http.MethodDelete, tt.url, tt.userID, tt.params)
			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestHandler_GetMyBookings(t *testing.T) {
	uid := int64(1)

	t.Run("success with query params", func(t *testing.T) {
		repo := &mockBooker{
			getUserBookingsFn: func(ctx context.Context, filter model.BookingFilter) ([]model.BookingResponse, error) {
				return []model.BookingResponse{{ID: 1}}, nil
			},
		}
		h := NewHandler(repo)

		w := performTestRequest(h.GetMyBookings, http.MethodGet, "/bookings?status=active&date=2026-07-31&limit=5&offset=2", &uid, nil)
		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("unauthorized", func(t *testing.T) {
		repo := &mockBooker{}
		h := NewHandler(repo)

		w := performTestRequest(h.GetMyBookings, http.MethodGet, "/bookings", nil, nil)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}
	})

	t.Run("repo error", func(t *testing.T) {
		repo := &mockBooker{
			getUserBookingsFn: func(ctx context.Context, filter model.BookingFilter) ([]model.BookingResponse, error) {
				return nil, errors.New("error")
			},
		}
		h := NewHandler(repo)

		w := performTestRequest(h.GetMyBookings, http.MethodGet, "/bookings", &uid, nil)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}
	})
}
