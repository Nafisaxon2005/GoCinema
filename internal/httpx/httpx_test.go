package httpx

import (
	"encoding/json"
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

func newTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c, w
}

func TestRespondError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantBody   string // пусто = не проверяем текст (для internal error)
	}{
		{
			name:       "ErrNotFound -> 404",
			err:        model.ErrNotFound,
			wantStatus: http.StatusNotFound,
			wantBody:   model.ErrNotFound.Error(),
		},
		{
			name:       "ErrInvalid -> 400",
			err:        model.ErrInvalid,
			wantStatus: http.StatusBadRequest,
			wantBody:   model.ErrInvalid.Error(),
		},
		{
			name:       "ErrForbidden -> 403",
			err:        model.ErrForbidden,
			wantStatus: http.StatusForbidden,
			wantBody:   model.ErrForbidden.Error(),
		},
		{
			name:       "ErrUnauthorized -> 401",
			err:        model.ErrUnauthorized,
			wantStatus: http.StatusUnauthorized,
			wantBody:   model.ErrUnauthorized.Error(),
		},
		{
			name:       "ErrAlreadyExists -> 409",
			err:        model.ErrAlreadyExists,
			wantStatus: http.StatusConflict,
			wantBody:   model.ErrAlreadyExists.Error(),
		},
		{
			name:       "ErrSeatTaken -> 409",
			err:        model.ErrSeatTaken,
			wantStatus: http.StatusConflict,
			wantBody:   model.ErrSeatTaken.Error(),
		},
		{
			name:       "неизвестная ошибка -> 500, сообщение скрыто",
			err:        errors.New("что-то сломалось в БД"),
			wantStatus: http.StatusInternalServerError,
			wantBody:   "internal error",
		},
		{
			name:       "обёрнутая ошибка распознаётся через errors.Is",
			err:        errorsWrap(model.ErrNotFound),
			wantStatus: http.StatusNotFound,
			wantBody:   errorsWrap(model.ErrNotFound).Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := newTestContext()

			RespondError(c, tt.err)

			if w.Code != tt.wantStatus {
				t.Errorf("статус = %d, ожидалось %d", w.Code, tt.wantStatus)
			}

			var body errorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("не удалось распарсить тело ответа: %v", err)
			}
			if body.Error != tt.wantBody {
				t.Errorf("Error = %q, ожидалось %q", body.Error, tt.wantBody)
			}
		})
	}
}
func errorsWrap(err error) error {
	return &wrappedErr{inner: err}
}

type wrappedErr struct{ inner error }

func (e *wrappedErr) Error() string { return "context: " + e.inner.Error() }
func (e *wrappedErr) Unwrap() error { return e.inner }

func TestRespondOK(t *testing.T) {
	c, w := newTestContext()

	RespondOK(c, map[string]string{"status": "ok"})

	if w.Code != http.StatusOK {
		t.Errorf("статус = %d, ожидалось %d", w.Code, http.StatusOK)
	}
	if !containsSubstring(w.Body.String(), `"status":"ok"`) {
		t.Errorf("тело ответа не содержит ожидаемых данных: %s", w.Body.String())
	}
}

func TestRespondCreated(t *testing.T) {
	c, w := newTestContext()

	RespondCreated(c, map[string]int{"id": 1})

	if w.Code != http.StatusCreated {
		t.Errorf("статус = %d, ожидалось %d", w.Code, http.StatusCreated)
	}
}

func TestRespondJSON(t *testing.T) {
	c, w := newTestContext()

	RespondJSON(c, http.StatusTeapot, map[string]string{"foo": "bar"})

	if w.Code != http.StatusTeapot {
		t.Errorf("статус = %d, ожидалось %d", w.Code, http.StatusTeapot)
	}
}

func containsSubstring(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
