package repository

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/raxima/seatpicker/internal/middleware"
	"github.com/raxima/seatpicker/internal/model"
)

const uniqueViolationCode = "23505"

type PgUserRepo struct {
	db     DBTX
	logger *slog.Logger
}

func NewPgUserRepo(db DBTX, logger *slog.Logger) *PgUserRepo {
	return &PgUserRepo{db: db, logger: logger}
}

func (r *PgUserRepo) Create(ctx context.Context, u *model.User) (int64, error) {
	log := r.logger.With(
		"layer", "repository",
		"op", "Create",
		"request_id", middleware.RequestIDFromContext(ctx),
	)

	const q = `
		INSERT INTO users (login, password_hash, role)
		VALUES ($1, $2, $3)
		RETURNING id`

	var id int64
	err := r.db.QueryRow(ctx, q, u.Login, u.PasswordHash, u.Role).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			log.Warn("логин уже занят", "login", u.Login)
			return 0, model.ErrAlreadyExists
		}
		log.Error("ошибка при создании пользователя", "login", u.Login, "error", err)
		return 0, err
	}

	log.Info("пользователь создан в БД", "user_id", id, "login", u.Login)
	return id, nil
}

func (r *PgUserRepo) GetByLogin(ctx context.Context, login string) (*model.User, error) {
	log := r.logger.With(
		"layer", "repository",
		"op", "GetByLogin",
		"request_id", middleware.RequestIDFromContext(ctx),
	)

	const q = `
		SELECT id, login, password_hash, role
		FROM users
		WHERE login = $1`

	var u model.User
	err := r.db.QueryRow(ctx, q, login).Scan(&u.ID, &u.Login, &u.PasswordHash, &u.Role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Info("пользователь не найден", "login", login)
			return nil, model.ErrNotFound
		}
		log.Error("ошибка при поиске пользователя по логину", "login", login, "error", err)
		return nil, err
	}
	return &u, nil
}

func (r *PgUserRepo) GetByID(ctx context.Context, id int64) (*model.User, error) {
	log := r.logger.With(
		"layer", "repository",
		"op", "GetByID",
		"request_id", middleware.RequestIDFromContext(ctx),
	)

	const q = `
		SELECT id, login, password_hash, role
		FROM users
		WHERE id = $1`

	var u model.User
	err := r.db.QueryRow(ctx, q, id).Scan(&u.ID, &u.Login, &u.PasswordHash, &u.Role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Info("пользователь не найден", "user_id", id)
			return nil, model.ErrNotFound
		}
		log.Error("ошибка при поиске пользователя по id", "user_id", id, "error", err)
		return nil, err
	}
	return &u, nil
}
