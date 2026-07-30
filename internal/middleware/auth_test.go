package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/raxima/seatpicker/internal/jwtutil"
)

func generateTestToken(t *testing.T, secret []byte, userID int64, expiresAt time.Time, signingMethod jwt.SigningMethod) string {
	t.Helper()
	claims := &jwtutil.Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token := jwt.NewWithClaims(signingMethod, claims)
	signed, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("не удалось подписать тестовый токен: %v", err)
	}
	return signed
}

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := []byte("test-secret")

	newTestRouter := func() *gin.Engine {
		r := gin.New()
		r.GET("/protected", AuthMiddleware(secret), func(c *gin.Context) {
			userID, _ := c.Get("userID")
			c.JSON(http.StatusOK, gin.H{"userID": userID})
		})
		return r
	}

	t.Run("валидный токен пропускает запрос", func(t *testing.T) {
		r := newTestRouter()
		token := generateTestToken(t, secret, 42, time.Now().Add(time.Hour), jwt.SigningMethodHS256)

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("статус = %d, ожидалось %d, тело: %s", w.Code, http.StatusOK, w.Body.String())
		}
	})

	t.Run("отсутствует заголовок Authorization", func(t *testing.T) {
		r := newTestRouter()

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("статус = %d, ожидалось %d", w.Code, http.StatusUnauthorized)
		}
	})

	t.Run("заголовок без Bearer", func(t *testing.T) {
		r := newTestRouter()

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Basic something")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("статус = %d, ожидалось %d", w.Code, http.StatusUnauthorized)
		}
	})

	t.Run("заголовок без токена после Bearer", func(t *testing.T) {
		r := newTestRouter()

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("статус = %d, ожидалось %d", w.Code, http.StatusUnauthorized)
		}
	})

	t.Run("истёкший токен", func(t *testing.T) {
		r := newTestRouter()
		token := generateTestToken(t, secret, 1, time.Now().Add(-time.Hour), jwt.SigningMethodHS256)

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("статус = %d, ожидалось %d", w.Code, http.StatusUnauthorized)
		}
	})

	t.Run("неверный секрет", func(t *testing.T) {
		r := newTestRouter()
		token := generateTestToken(t, []byte("wrong-secret"), 1, time.Now().Add(time.Hour), jwt.SigningMethodHS256)

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("статус = %d, ожидалось %d", w.Code, http.StatusUnauthorized)
		}
	})

	t.Run("мусорный токен", func(t *testing.T) {
		r := newTestRouter()

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer not-a-jwt")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("статус = %d, ожидалось %d", w.Code, http.StatusUnauthorized)
		}
	})
}
