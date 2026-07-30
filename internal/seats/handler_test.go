package seats

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/raxima/seatpicker/internal/model"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ---- fake Booker ----

type fakeBooker struct {
	bookSeatErr        error
	bookSeatID         int64
	cancelErr          error
	getShowSeats       []model.Seat
	getShowSeatsErr    error
	getUserBookings    []model.BookingResponse
	getUserBookingsErr error
}

func (f *fakeBooker) BookSeat(ctx context.Context, showID, seatID, userID int64) (int64, error) {
	if f.bookSeatErr != nil {
		return 0, f.bookSeatErr
	}
	return f.bookSeatID, nil
}

func (f *fakeBooker) CancelBooking(ctx context.Context, bookingID, userID int64) error {
	return f.cancelErr
}

func (f *fakeBooker) GetShowSeats(ctx context.Context, showID int64) ([]model.Seat, error) {
	return f.getShowSeats, f.getShowSeatsErr
}

func (f *fakeBooker) GetUserBookings(ctx context.Context, filter model.BookingFilter) ([]model.BookingResponse, error) {
	return f.getUserBookings, f.getUserBookingsErr
}

func newRouterWithUser(h *Handler, userID int64, setUser bool) *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if setUser {
			c.Set("userID", userID)
		}
		c.Next()
	})
	r.POST("/shows/:id/seats/:seatId/book", h.Book)
	r.GET("/shows/:id/seats", h.GetSeats)
	r.DELETE("/bookings/:bookingId", h.Cancel)
	r.GET("/bookings", h.GetMyBookings)
	return r
}

func TestHandler_Book(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		setUser    bool
		bookErr    error
		wantStatus int
	}{
		{name: "успешное бронирование", target: "/shows/1/seats/2/book", setUser: true, wantStatus: http.StatusCreated},
		{name: "невалидный show id", target: "/shows/abc/seats/2/book", setUser: true, wantStatus: http.StatusBadRequest},
		{name: "невалидный seat id", target: "/shows/1/seats/abc/book", setUser: true, wantStatus: http.StatusBadRequest},
		{name: "нет userID", target: "/shows/1/seats/2/book", setUser: false, wantStatus: http.StatusUnauthorized},
		{name: "место занято", target: "/shows/1/seats/2/book", setUser: true, bookErr: ErrSeatTaken, wantStatus: http.StatusConflict},
		{name: "сеанс недоступен", target: "/shows/1/seats/2/book", setUser: true, bookErr: ErrShowNotAvailable, wantStatus: http.StatusBadRequest},
		{name: "внутренняя ошибка", target: "/shows/1/seats/2/book", setUser: true, bookErr: context.DeadlineExceeded, wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fb := &fakeBooker{bookSeatErr: tt.bookErr, bookSeatID: 100}
			h := NewHandler(fb)
			r := newRouterWithUser(h, 1, tt.setUser)

			req := httptest.NewRequest(http.MethodPost, tt.target, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("статус = %d, ожидалось %d, тело: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestHandler_GetSeats(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		seats      []model.Seat
		repoErr    error
		wantStatus int
	}{
		{name: "успешный список", target: "/shows/1/seats", seats: []model.Seat{{ID: 1}, {ID: 2}}, wantStatus: http.StatusOK},
		{name: "невалидный show id", target: "/shows/abc/seats", wantStatus: http.StatusBadRequest},
		{name: "ошибка репозитория", target: "/shows/1/seats", repoErr: context.DeadlineExceeded, wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fb := &fakeBooker{getShowSeats: tt.seats, getShowSeatsErr: tt.repoErr}
			h := NewHandler(fb)
			r := newRouterWithUser(h, 1, true)

			req := httptest.NewRequest(http.MethodGet, tt.target, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("статус = %d, ожидалось %d, тело: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestHandler_Cancel(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		setUser    bool
		cancelErr  error
		wantStatus int
	}{
		{name: "успешная отмена", target: "/bookings/1", setUser: true, wantStatus: http.StatusNoContent},
		{name: "невалидный booking id", target: "/bookings/abc", setUser: true, wantStatus: http.StatusBadRequest},
		{name: "нет userID", target: "/bookings/1", setUser: false, wantStatus: http.StatusUnauthorized},
		{name: "бронь не найдена", target: "/bookings/1", setUser: true, cancelErr: ErrBookingNotFound, wantStatus: http.StatusNotFound},
		{name: "чужая бронь", target: "/bookings/1", setUser: true, cancelErr: ErrForbidden, wantStatus: http.StatusForbidden},
		{name: "внутренняя ошибка", target: "/bookings/1", setUser: true, cancelErr: context.DeadlineExceeded, wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fb := &fakeBooker{cancelErr: tt.cancelErr}
			h := NewHandler(fb)
			r := newRouterWithUser(h, 1, tt.setUser)

			req := httptest.NewRequest(http.MethodDelete, tt.target, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("статус = %d, ожидалось %d, тело: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestHandler_GetMyBookings(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		setUser    bool
		bookings   []model.BookingResponse
		repoErr    error
		wantStatus int
	}{
		{name: "успешный список без фильтров", target: "/bookings", setUser: true, bookings: []model.BookingResponse{{ID: 1}}, wantStatus: http.StatusOK},
		{name: "с фильтрами status/date/limit/offset", target: "/bookings?status=booked&date=2026-01-01&limit=5&offset=10", setUser: true, wantStatus: http.StatusOK},
		{name: "нет userID", target: "/bookings", setUser: false, wantStatus: http.StatusUnauthorized},
		{name: "ошибка репозитория", target: "/bookings", setUser: true, repoErr: context.DeadlineExceeded, wantStatus: http.StatusInternalServerError},
		{name: "невалидный limit игнорируется (дефолт)", target: "/bookings?limit=abc", setUser: true, wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fb := &fakeBooker{getUserBookings: tt.bookings, getUserBookingsErr: tt.repoErr}
			h := NewHandler(fb)
			r := newRouterWithUser(h, 1, tt.setUser)

			req := httptest.NewRequest(http.MethodGet, tt.target, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("статус = %d, ожидалось %d, тело: %s", w.Code, tt.wantStatus, w.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				var got []model.BookingResponse
				if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
					t.Errorf("не удалось распарсить тело: %v", err)
				}
			}
		})
	}
}
