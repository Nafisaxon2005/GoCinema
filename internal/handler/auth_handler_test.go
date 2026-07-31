package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/raxima/seatpicker/internal/model"
	_ "github.com/raxima/seatpicker/internal/repository"
	"github.com/raxima/seatpicker/internal/service"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type fakeUserRepo struct {
	byLogin map[string]*model.User
	byID    map[int64]*model.User
	nextID  int64
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{byLogin: make(map[string]*model.User), byID: make(map[int64]*model.User)}
}

func (r *fakeUserRepo) Create(ctx context.Context, u *model.User) (int64, error) {
	if _, exists := r.byLogin[u.Login]; exists {
		return 0, model.ErrAlreadyExists
	}
	r.nextID++
	u.ID = r.nextID
	cp := *u
	r.byLogin[u.Login] = &cp
	r.byID[u.ID] = &cp
	return u.ID, nil
}

func (r *fakeUserRepo) GetByLogin(ctx context.Context, login string) (*model.User, error) {
	u, ok := r.byLogin[login]
	if !ok {
		return nil, model.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (r *fakeUserRepo) GetByID(ctx context.Context, id int64) (*model.User, error) {
	u, ok := r.byID[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

type fakeRefreshTokenRepo struct {
	byHash map[string]*model.RefreshToken
	nextID int64
}

func newFakeRefreshTokenRepo() *fakeRefreshTokenRepo {
	return &fakeRefreshTokenRepo{byHash: make(map[string]*model.RefreshToken)}
}

func (r *fakeRefreshTokenRepo) Create(ctx context.Context, rt *model.RefreshToken) (int64, error) {
	r.nextID++
	rt.ID = r.nextID
	cp := *rt
	r.byHash[rt.TokenHash] = &cp
	return rt.ID, nil
}

func (r *fakeRefreshTokenRepo) GetByHash(ctx context.Context, hash string) (*model.RefreshToken, error) {
	rt, ok := r.byHash[hash]
	if !ok {
		return nil, model.ErrNotFound
	}
	cp := *rt
	return &cp, nil
}

func (r *fakeRefreshTokenRepo) Revoke(ctx context.Context, id int64) error {
	for _, rt := range r.byHash {
		if rt.ID == id {
			now := time.Now()
			rt.RevokedAt = &now
			return nil
		}
	}
	return model.ErrNotFound
}

func (r *fakeRefreshTokenRepo) RevokeAllForUser(ctx context.Context, userID int64) error {
	return nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

func newTestAuthHandler() *AuthHandler {
	users := newFakeUserRepo()
	refreshTokens := newFakeRefreshTokenRepo()
	cfg := service.AuthConfig{JWTSecret: []byte("test-secret"), AccessTTL: 15 * time.Minute, RefreshTTL: 24 * time.Hour}
	authService := service.NewAuthService(users, refreshTokens, cfg, testLogger())
	return NewAuthHandler(authService, testLogger())
}

func doJSONRequest(h gin.HandlerFunc, method, path string, body any) *httptest.ResponseRecorder {
	r := gin.New()
	switch method {
	case http.MethodPost:
		r.POST(path, h)
	}

	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}

	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestAuthHandler_Register(t *testing.T) {
	tests := []struct {
		name       string
		body       any
		wantStatus int
	}{
		{
			name:       "успешная регистрация",
			body:       map[string]any{"login": "alice", "password": "12345678", "role": "viewer"},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "невалидное тело — отсутствует пароль",
			body:       map[string]any{"login": "bob", "role": "viewer"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "невалидная роль по бизнес-правилам",
			body:       map[string]any{"login": "carol", "password": "12345678", "role": "hacker"},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestAuthHandler()
			w := doJSONRequest(h.Register, http.MethodPost, "/auth/register", tt.body)

			if w.Code != tt.wantStatus {
				t.Errorf("статус = %d, ожидалось %d, тело: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestAuthHandler_Login(t *testing.T) {
	h := newTestAuthHandler()

	doJSONRequest(h.Register, http.MethodPost, "/auth/register",
		map[string]any{"login": "dave", "password": "12345678", "role": "viewer"})

	tests := []struct {
		name       string
		body       any
		wantStatus int
	}{
		{
			name:       "успешный вход",
			body:       map[string]any{"login": "dave", "password": "12345678"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "неверный пароль",
			body:       map[string]any{"login": "dave", "password": "wrongpass"},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "невалидное тело",
			body:       map[string]any{"login": "dave"},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := doJSONRequest(h.Login, http.MethodPost, "/auth/login", tt.body)
			if w.Code != tt.wantStatus {
				t.Errorf("статус = %d, ожидалось %d, тело: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestAuthHandler_RefreshAndLogout(t *testing.T) {
	h := newTestAuthHandler()

	doJSONRequest(h.Register, http.MethodPost, "/auth/register",
		map[string]any{"login": "erin", "password": "12345678", "role": "viewer"})
	loginResp := doJSONRequest(h.Login, http.MethodPost, "/auth/login",
		map[string]any{"login": "erin", "password": "12345678"})

	var pair struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(loginResp.Body.Bytes(), &pair); err != nil {
		t.Fatalf("не удалось распарсить ответ логина: %v", err)
	}

	t.Run("refresh с валидным токеном", func(t *testing.T) {
		w := doJSONRequest(h.Refresh, http.MethodPost, "/auth/refresh",
			map[string]any{"refresh_token": pair.RefreshToken})
		if w.Code != http.StatusOK {
			t.Errorf("статус = %d, ожидалось %d, тело: %s", w.Code, http.StatusOK, w.Body.String())
		}
	})

	t.Run("refresh с невалидным токеном", func(t *testing.T) {
		w := doJSONRequest(h.Refresh, http.MethodPost, "/auth/refresh",
			map[string]any{"refresh_token": "garbage"})
		if w.Code != http.StatusUnauthorized {
			t.Errorf("статус = %d, ожидалось %d", w.Code, http.StatusUnauthorized)
		}
	})

	t.Run("logout", func(t *testing.T) {
		w := doJSONRequest(h.Logout, http.MethodPost, "/auth/logout",
			map[string]any{"refresh_token": pair.RefreshToken})
		if w.Code != http.StatusOK {
			t.Errorf("статус = %d, ожидалось %d, тело: %s", w.Code, http.StatusOK, w.Body.String())
		}
	})
}
