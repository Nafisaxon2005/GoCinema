package middleware

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRequestID(t *testing.T) {
	r := gin.New()
	var capturedFromContext string

	r.Use(RequestID())
	r.GET("/test", func(c *gin.Context) {
		capturedFromContext = RequestIDFromContext(c.Request.Context())
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	t.Run("устанавливает заголовок X-Request-ID", func(t *testing.T) {
		if w.Header().Get("X-Request-ID") == "" {
			t.Error("заголовок X-Request-ID не должен быть пустым")
		}
	})

	t.Run("request_id доступен через context внутри хендлера", func(t *testing.T) {
		if capturedFromContext == "" {
			t.Error("RequestIDFromContext вернул пустую строку внутри хендлера")
		}
		if capturedFromContext != w.Header().Get("X-Request-ID") {
			t.Error("request_id из контекста должен совпадать с заголовком ответа")
		}
	})

	t.Run("два запроса получают разные request_id", func(t *testing.T) {
		req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
		w2 := httptest.NewRecorder()
		r.ServeHTTP(w2, req2)

		if w.Header().Get("X-Request-ID") == w2.Header().Get("X-Request-ID") {
			t.Error("разные запросы не должны получать одинаковый request_id")
		}
	})
}

func TestRequestIDFromContext_empty(t *testing.T) {
	if got := RequestIDFromContext(context.Background()); got != "" {
		t.Errorf("для пустого контекста ожидалась пустая строка, получено %q", got)
	}
}

func TestLogger(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	r := gin.New()
	r.Use(RequestID())
	r.Use(Logger(logger))
	r.GET("/ok", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	r.GET("/fail", func(c *gin.Context) {
		c.Status(http.StatusInternalServerError)
	})

	t.Run("не паникует на успешном запросе", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ok", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("код ответа = %d, ожидалось %d", w.Code, http.StatusOK)
		}
	})

	t.Run("не паникует на запросе с ошибкой", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/fail", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("код ответа = %d, ожидалось %d", w.Code, http.StatusInternalServerError)
		}
	})
}
