package service

import (
	"context"

	"golang.org/x/crypto/bcrypt"

	"github.com/raxima/seatpicker/internal/model"
	"github.com/raxima/seatpicker/internal/repository"
)

const minPasswordLength = 8

type AuthService struct {
	users repository.UserRepo
}

func NewAuthService(users repository.UserRepo) *AuthService {
	return &AuthService{users: users}
}

// RegisterInput — то, что приходит из handler после биндинга JSON.
type RegisterInput struct {
	Login    string
	Password string
	Role     model.Role
}

func (s *AuthService) Register(ctx context.Context, in RegisterInput) (*model.User, error) {
	if len(in.Password) < minPasswordLength {
		return nil, model.ErrInvalid
	}
	if in.Login == "" {
		return nil, model.ErrInvalid
	}
	switch in.Role {
	case model.RoleViewer, model.RoleOrganizer, model.RoleAdmin:
	default:
		return nil, model.ErrInvalid
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	u := &model.User{
		Login:        in.Login,
		PasswordHash: string(hash),
		Role:         in.Role,
	}

	id, err := s.users.Create(ctx, u)
	if err != nil {
		return nil, err
	}
	u.ID = id
	return u, nil
}
