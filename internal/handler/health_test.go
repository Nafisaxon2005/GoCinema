package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakePinger struct {
	err error
}

func (p *fakePinger) Ping(ctx context.Context) error {
	return p.err
}

func TestHealthHandler_Health(t *testing.T) {
	tests := []struct {
		name       string
		dbErr      error
		wantStatus int
	}{
		{
			name:       "БД доступна",
			dbErr:      nil,
			wantStatus: http.StatusOK,
		},
		{
			name:       "БД недоступна",
			dbErr:      errors.New("connection refused"),
			wantStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHealthHandler(&fakePinger{err: tt.dbErr}, testLogger())

			r := gin.New()
			r.GET("/health", h.Health)

			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("статус = %d, ожидалось %d, тело: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}
