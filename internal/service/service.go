package service

import (
	"context"
	"log/slog"

	"golang.org/x/crypto/bcrypt"

	"github.com/raxima/seatpicker/internal/model"
	"github.com/raxima/seatpicker/internal/repository"
)

const minPasswordLength = 8

type AuthService struct {
	users  repository.UserRepo
	logger *slog.Logger
}

func NewAuthService(users repository.UserRepo, logger *slog.Logger) *AuthService {
	return &AuthService{users: users, logger: logger}
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
