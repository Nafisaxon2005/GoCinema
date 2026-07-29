package service

import (
	"context"
	"time"

	"github.com/raxima/seatpicker/internal/jwtutil"
	"github.com/raxima/seatpicker/internal/model"
	"golang.org/x/crypto/bcrypt"
)

// TokenPair — пара токенов, отдаётся клиенту после Login/Refresh.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

// Логин проверяет логин/пароль и выдаёт пару токенов.
func (s *AuthService) Login(ctx context.Context, login, password string) (*TokenPair, error) {
	u, err := s.users.GetByLogin(ctx, login)
	if err != nil {
		// Не различаем "нет пользователя" и "неверный пароль" — иначе можно
		// перебором логинов узнать, какие из них зарегистрированы.
		return nil, model.ErrUnauthorized
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, model.ErrUnauthorized
	}

	return s.issueTokenPair(ctx, u)
}

// Refresh проверяет refresh-токен, отзывает его (ротация) и выдаёт новую пару.
func (s *AuthService) Refresh(ctx context.Context, rawRefreshToken string) (*TokenPair, error) {
	hash := jwtutil.HashRefreshToken(rawRefreshToken)

	rt, err := s.refreshTokens.GetByHash(ctx, hash)
	if err != nil {
		return nil, model.ErrUnauthorized
	}
	if rt.RevokedAt != nil || rt.ExpiresAt.Before(time.Now()) {
		return nil, model.ErrUnauthorized
	}

	u, err := s.users.GetByID(ctx, rt.UserID)
	if err != nil {
		return nil, model.ErrUnauthorized
	}

	// Ротация: старый refresh отзываем перед выдачей нового.
	if err := s.refreshTokens.Revoke(ctx, rt.ID); err != nil {
		return nil, err
	}

	return s.issueTokenPair(ctx, u)
}

// Logout отзывает refresh-токен. Идемпотентен: если токен не найден
// (уже отозван/невалиден), просто ничего не делает.
func (s *AuthService) Logout(ctx context.Context, rawRefreshToken string) error {
	hash := jwtutil.HashRefreshToken(rawRefreshToken)

	rt, err := s.refreshTokens.GetByHash(ctx, hash)
	if err != nil {
		if err == model.ErrNotFound {
			return nil
		}
		return err
	}

	return s.refreshTokens.Revoke(ctx, rt.ID)
}

// issueTokenPair генерирует access+refresh и сохраняет хэш refresh в БД.
func (s *AuthService) issueTokenPair(ctx context.Context, u *model.User) (*TokenPair, error) {
	access, err := jwtutil.GenerateAccess(u.ID, u.Role, s.cfg.JWTSecret, s.cfg.AccessTTL)
	if err != nil {
		return nil, err
	}

	rawRefresh, hash, err := jwtutil.NewRefreshToken()
	if err != nil {
		return nil, err
	}

	_, err = s.refreshTokens.Create(ctx, &model.RefreshToken{
		UserID:    u.ID,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(s.cfg.RefreshTTL),
	})
	if err != nil {
		return nil, err
	}

	return &TokenPair{AccessToken: access, RefreshToken: rawRefresh}, nil
}
