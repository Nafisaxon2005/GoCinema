package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/raxima/seatpicker/internal/model"
	"github.com/raxima/seatpicker/internal/repository"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// --- мок AdminService ---

type mockAdminService struct {
	getStatsFn       func(ctx context.Context, from, to time.Time) ([]model.ShowSalesStat, error)
	listShowsFn      func(ctx context.Context, f repository.AdminShowFilter) ([]model.Show, int, error)
	moderateShowFn   func(ctx context.Context, showID int64, action string) error
	refundBookingFn  func(ctx context.Context, bookingID int64, reason string) error
	listRefundsFn    func(ctx context.Context, limit, offset int) ([]model.RefundResponse, error)
	listStaleHoldsFn func(ctx context.Context, olderThan time.Duration) ([]model.HeldSeatResponse, error)
}

func (m *mockAdminService) GetStats(ctx context.Context, from, to time.Time) ([]model.ShowSalesStat, error) {
	return m.getStatsFn(ctx, from, to)
}

func (m *mockAdminService) ListShows(ctx context.Context, f repository.AdminShowFilter) ([]model.Show, int, error) {
	return m.listShowsFn(ctx, f)
}

func (m *mockAdminService) ModerateShow(ctx context.Context, showID int64, action string) error {
	return m.moderateShowFn(ctx, showID, action)
}

func (m *mockAdminService) RefundBooking(ctx context.Context, bookingID int64, reason string) error {
	if m.refundBookingFn != nil {
		return m.refundBookingFn(ctx, bookingID, reason)
	}
	return nil
}

func (m *mockAdminService) ListRefunds(ctx context.Context, limit, offset int) ([]model.RefundResponse, error) {
	if m.listRefundsFn != nil {
		return m.listRefundsFn(ctx, limit, offset)
	}
	return nil, nil
}

func (m *mockAdminService) ListStaleHolds(ctx context.Context, olderThan time.Duration) ([]model.HeldSeatResponse, error) {
	if m.listStaleHoldsFn != nil {
		return m.listStaleHoldsFn(ctx, olderThan)
	}
	return nil, nil
}

func performRequest(handlerFunc gin.HandlerFunc, method, url string, body []byte, params gin.Params) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var reqBody *bytes.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	} else {
		reqBody = bytes.NewReader([]byte{})
	}

	req := httptest.NewRequest(method, url, reqBody)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	c.Request = req
	c.Params = params

	handlerFunc(c)
	return w
}

// --- тесты GetStats ---

func TestAdminHandler_GetStats(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		serviceErr error
		wantStatus int
	}{
		{
			name:       "default period succeeds",
			url:        "/admin/stats",
			wantStatus: http.StatusOK,
		},
		{
			name:       "explicit valid period succeeds",
			url:        "/admin/stats?from=2026-07-01&to=2026-07-31",
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid from date returns 400",
			url:        "/admin/stats?from=not-a-date",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid to date returns 400",
			url:        "/admin/stats?to=not-a-date",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "service error returns invalid",
			url:        "/admin/stats",
			serviceErr: model.ErrInvalid,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockAdminService{
				getStatsFn: func(ctx context.Context, from, to time.Time) ([]model.ShowSalesStat, error) {
					if tt.serviceErr != nil {
						return nil, tt.serviceErr
					}
					return []model.ShowSalesStat{{ShowID: 1, Sold: 5, Total: 10, Revenue: 500}}, nil
				},
			}
			h := NewAdminHandler(svc)

			w := performRequest(h.GetStats, http.MethodGet, tt.url, nil, nil)
			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d, body=%s", tt.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

// --- тесты GetAllShows ---

func TestAdminHandler_GetAllShows(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		serviceErr error
		wantStatus int
	}{
		{
			name:       "no filters succeeds",
			url:        "/admin/shows",
			wantStatus: http.StatusOK,
		},
		{
			name:       "with filters succeeds",
			url:        "/admin/shows?search=dune&venue=Hall1&organizer_id=4&status=draft&from=2026-07-01&to=2026-07-31&page=2&page_size=10",
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid organizer_id returns 400",
			url:        "/admin/shows?organizer_id=not-a-number",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid from date returns 400",
			url:        "/admin/shows?from=bad-date",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid to date returns 400",
			url:        "/admin/shows?to=bad-date",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "service error propagates",
			url:        "/admin/shows",
			serviceErr: model.ErrInvalid,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockAdminService{
				listShowsFn: func(ctx context.Context, f repository.AdminShowFilter) ([]model.Show, int, error) {
					if tt.serviceErr != nil {
						return nil, 0, tt.serviceErr
					}
					return []model.Show{{ID: 1, Title: "Дюна 2"}}, 1, nil
				},
			}
			h := NewAdminHandler(svc)

			w := performRequest(h.GetAllShows, http.MethodGet, tt.url, nil, nil)
			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d, body=%s", tt.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

// --- тесты ModerateShow ---

func TestAdminHandler_ModerateShow(t *testing.T) {
	tests := []struct {
		name       string
		idParam    string
		body       string
		serviceErr error
		wantStatus int
	}{
		{
			name:       "approve succeeds",
			idParam:    "1",
			body:       `{"action":"approve"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "reject succeeds",
			idParam:    "1",
			body:       `{"action":"reject"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid id param returns 400",
			idParam:    "not-a-number",
			body:       `{"action":"approve"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid action in body returns 400",
			idParam:    "1",
			body:       `{"action":"delete"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing action field returns 400",
			idParam:    "1",
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "malformed json returns 400",
			idParam:    "1",
			body:       `not-json`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "service error propagates",
			idParam:    "1",
			body:       `{"action":"approve"}`,
			serviceErr: model.ErrInvalid,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "not found propagates as 404",
			idParam:    "999",
			body:       `{"action":"approve"}`,
			serviceErr: model.ErrNotFound,
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockAdminService{
				moderateShowFn: func(ctx context.Context, showID int64, action string) error {
					return tt.serviceErr
				},
			}
			h := NewAdminHandler(svc)

			params := gin.Params{{Key: "id", Value: tt.idParam}}
			w := performRequest(h.ModerateShow, http.MethodPut, "/admin/shows/"+tt.idParam+"/moderate", []byte(tt.body), params)
			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d, body=%s", tt.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

// --- тесты RefundBooking (C-04) ---

func TestAdminHandler_RefundBooking(t *testing.T) {
	tests := []struct {
		name       string
		idParam    string
		url        string
		serviceErr error
		wantStatus int
	}{
		{
			name:       "valid refund succeeds",
			idParam:    "1",
			url:        "/admin/bookings/1/refund?reason=test",
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid id returns 400",
			idParam:    "not-a-number",
			url:        "/admin/bookings/not-a-number/refund?reason=test",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing reason returns 400",
			idParam:    "1",
			url:        "/admin/bookings/1/refund",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "booking not found propagates",
			idParam:    "999",
			url:        "/admin/bookings/999/refund?reason=test",
			serviceErr: model.ErrNotFound,
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockAdminService{
				refundBookingFn: func(ctx context.Context, bookingID int64, reason string) error {
					return tt.serviceErr
				},
			}

			h := NewAdminHandler(svc)

			params := gin.Params{
				{Key: "id", Value: tt.idParam},
			}

			w := performRequest(
				h.RefundBooking,
				http.MethodGet,
				tt.url,
				nil,
				params,
			)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d, body=%s", tt.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

// --- тесты ListRefunds (C-04) ---

func TestAdminHandler_ListRefunds(t *testing.T) {
	svc := &mockAdminService{
		listRefundsFn: func(ctx context.Context, limit, offset int) ([]model.RefundResponse, error) {
			return []model.RefundResponse{{BookingID: 1, Reason: "test"}}, nil
		},
	}
	h := NewAdminHandler(svc)

	w := performRequest(h.ListRefunds, http.MethodGet, "/admin/refunds?limit=10&offset=0", nil, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
}

// --- тесты ListHolds (C-06) ---

func TestAdminHandler_ListHolds(t *testing.T) {
	svc := &mockAdminService{
		listStaleHoldsFn: func(ctx context.Context, olderThan time.Duration) ([]model.HeldSeatResponse, error) {
			return []model.HeldSeatResponse{{SeatID: 1, ShowID: 1}}, nil
		},
	}
	h := NewAdminHandler(svc)

	w := performRequest(h.ListHolds, http.MethodGet, "/admin/holds?minutes=15", nil, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
}
