package service

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/raxima/seatpicker/internal/model"
)

type fakeUserRepo struct {
	byLogin   map[string]*model.User
	byID      map[int64]*model.User
	nextID    int64
	createErr error
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		byLogin: make(map[string]*model.User),
		byID:    make(map[int64]*model.User),
	}
}

func (r *fakeUserRepo) Create(ctx context.Context, u *model.User) (int64, error) {
	if r.createErr != nil {
		return 0, r.createErr
	}
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
	now := time.Now()
	for _, rt := range r.byHash {
		if rt.UserID == userID {
			rt.RevokedAt = &now
		}
	}
	return nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

func testAuthCfg() AuthConfig {
	return AuthConfig{
		JWTSecret:  []byte("test-secret"),
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 24 * time.Hour,
	}
}

func TestAuthService_Register(t *testing.T) {
	tests := []struct {
		name    string
		input   RegisterInput
		seed    func(*fakeUserRepo)
		wantErr error
	}{
		{
			name:  "успешная регистрация",
			input: RegisterInput{Login: "alice", Password: "12345678", Role: model.RoleViewer},
		},
		{
			name:    "слишком короткий пароль",
			input:   RegisterInput{Login: "bob", Password: "123", Role: model.RoleViewer},
			wantErr: model.ErrInvalid,
		},
		{
			name:    "пустой логин",
			input:   RegisterInput{Login: "", Password: "12345678", Role: model.RoleViewer},
			wantErr: model.ErrInvalid,
		},
		{
			name:    "недопустимая роль",
			input:   RegisterInput{Login: "carol", Password: "12345678", Role: model.Role("hacker")},
			wantErr: model.ErrInvalid,
		},
		{
			name:  "логин уже занят",
			input: RegisterInput{Login: "dave", Password: "12345678", Role: model.RoleViewer},
			seed: func(r *fakeUserRepo) {
				_, _ = r.Create(context.Background(), &model.User{Login: "dave", PasswordHash: "x", Role: model.RoleViewer})
			},
			wantErr: model.ErrAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users := newFakeUserRepo()
			if tt.seed != nil {
				tt.seed(users)
			}
			svc := NewAuthService(users, newFakeRefreshTokenRepo(), testAuthCfg(), testLogger())

			u, err := svc.Register(context.Background(), tt.input)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Fatalf("ожидалась ошибка %v, получено %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
			if u.Login != tt.input.Login {
				t.Errorf("Login = %q, ожидалось %q", u.Login, tt.input.Login)
			}
			if u.PasswordHash == tt.input.Password {
				t.Errorf("пароль не захеширован")
			}
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	users := newFakeUserRepo()
	hash, _ := bcrypt.GenerateFromPassword([]byte("correctpass"), bcrypt.DefaultCost)
	_, _ = users.Create(context.Background(), &model.User{
		Login:        "eve",
		PasswordHash: string(hash),
		Role:         model.RoleViewer,
	})

	tests := []struct {
		name     string
		login    string
		password string
		wantErr  error
	}{
		{name: "успешный вход", login: "eve", password: "correctpass"},
		{name: "неверный пароль", login: "eve", password: "wrongpass", wantErr: model.ErrUnauthorized},
		{name: "несуществующий логин", login: "ghost", password: "whatever", wantErr: model.ErrUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewAuthService(users, newFakeRefreshTokenRepo(), testAuthCfg(), testLogger())

			pair, err := svc.Login(context.Background(), tt.login, tt.password)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Fatalf("ожидалась ошибка %v, получено %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
			if pair.AccessToken == "" || pair.RefreshToken == "" {
				t.Error("токены не должны быть пустыми")
			}
		})
	}
}

func TestAuthService_RefreshAndLogout(t *testing.T) {
	users := newFakeUserRepo()
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass1234"), bcrypt.DefaultCost)
	_, _ = users.Create(context.Background(), &model.User{
		Login:        "frank",
		PasswordHash: string(hash),
		Role:         model.RoleViewer,
	})

	refreshRepo := newFakeRefreshTokenRepo()
	svc := NewAuthService(users, refreshRepo, testAuthCfg(), testLogger())

	pair, err := svc.Login(context.Background(), "frank", "pass1234")
	if err != nil {
		t.Fatalf("Login завершился с ошибкой: %v", err)
	}

	t.Run("refresh с валидным токеном выдаёт новую пару", func(t *testing.T) {
		newPair, err := svc.Refresh(context.Background(), pair.RefreshToken)
		if err != nil {
			t.Fatalf("Refresh завершился с ошибкой: %v", err)
		}
		if newPair.RefreshToken == pair.RefreshToken {
			t.Error("refresh-токен должен ротироваться (быть новым)")
		}

		t.Run("старый refresh-токен больше не работает", func(t *testing.T) {
			_, err := svc.Refresh(context.Background(), pair.RefreshToken)
			if err != model.ErrUnauthorized {
				t.Errorf("ожидалась ошибка %v, получено %v", model.ErrUnauthorized, err)
			}
		})
	})

	t.Run("refresh с невалидным токеном", func(t *testing.T) {
		_, err := svc.Refresh(context.Background(), "invalid-token")
		if err != model.ErrUnauthorized {
			t.Errorf("ожидалась ошибка %v, получено %v", model.ErrUnauthorized, err)
		}
	})

	t.Run("logout отзывает токен", func(t *testing.T) {
		pair2, _ := svc.Login(context.Background(), "frank", "pass1234")

		if err := svc.Logout(context.Background(), pair2.RefreshToken); err != nil {
			t.Fatalf("Logout завершился с ошибкой: %v", err)
		}

		_, err := svc.Refresh(context.Background(), pair2.RefreshToken)
		if err != model.ErrUnauthorized {
			t.Errorf("после logout refresh должен быть невалиден, получено %v", err)
		}
	})

	t.Run("logout несуществующего токена — идемпотентен", func(t *testing.T) {
		if err := svc.Logout(context.Background(), "never-existed"); err != nil {
			t.Errorf("Logout несуществующего токена не должен возвращать ошибку, получено %v", err)
		}
	})
}
