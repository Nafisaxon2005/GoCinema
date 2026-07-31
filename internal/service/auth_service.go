package service

import (
	"context"
	"log/slog"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/raxima/seatpicker/internal/jwtutil"
	"github.com/raxima/seatpicker/internal/model"
	"github.com/raxima/seatpicker/internal/repository"
)

const minPasswordLength = 8

type AuthConfig struct {
	JWTSecret  []byte
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

type AuthService struct {
	users         repository.UserRepo
	refreshTokens repository.RefreshTokenRepo
	cfg           AuthConfig
	logger        *slog.Logger
}

func NewAuthService(users repository.UserRepo, refreshTokens repository.RefreshTokenRepo, cfg AuthConfig, logger *slog.Logger) *AuthService {
	return &AuthService{users: users, refreshTokens: refreshTokens, cfg: cfg, logger: logger}
}

type RegisterInput struct {
	Login    string
	Password string
	Role     model.Role
}

func (s *AuthService) Register(ctx context.Context, in RegisterInput) (*model.User, error) {
	log := s.logger.With("layer", "service", "op", "Register")

	if len(in.Password) < minPasswordLength {
		log.Warn("слишком короткий пароль")
		return nil, model.ErrInvalid
	}
	if in.Login == "" {
		log.Warn("пустой логин")
		return nil, model.ErrInvalid
	}
	switch in.Role {
	case model.RoleViewer, model.RoleOrganizer, model.RoleAdmin:
	default:
		log.Warn("недопустимая роль", "role", in.Role)
		return nil, model.ErrInvalid
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Error("не удалось захешировать пароль", "error", err)
		return nil, err
	}

	u := &model.User{
		Login:        in.Login,
		PasswordHash: string(hash),
		Role:         in.Role,
	}

	id, err := s.users.Create(ctx, u)
	if err != nil {
		log.Error("не удалось создать пользователя в репозитории", "login", in.Login, "error", err)
		return nil, err
	}
	u.ID = id

	log.Info("пользователь зарегистрирован", "user_id", id, "login", in.Login, "role", in.Role)
	return u, nil
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

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

	if err := s.refreshTokens.Revoke(ctx, rt.ID); err != nil {
		return nil, err
	}

	return s.issueTokenPair(ctx, u)
}

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
