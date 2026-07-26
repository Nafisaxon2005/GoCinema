package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/raxima/seatpicker/internal/model"
)

const uniqueViolationCode = "23505"

type PgUserRepo struct {
	db *pgxpool.Pool
}

func NewPgUserRepo(db *pgxpool.Pool) *PgUserRepo {
	return &PgUserRepo{db: db}
}

func (r *PgUserRepo) Create(ctx context.Context, u *model.User) (int64, error) {
	const q = `
		INSERT INTO users (login, password_hash, role)
		VALUES ($1, $2, $3)
		RETURNING id`

	var id int64
	err := r.db.QueryRow(ctx, q, u.Login, u.PasswordHash, u.Role).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return 0, model.ErrAlreadyExists
		}
		return 0, err
	}
	return id, nil
}

func (r *PgUserRepo) GetByLogin(ctx context.Context, login string) (*model.User, error) {
	const q = `
		SELECT id, login, password_hash, role
		FROM users
		WHERE login = $1`

	var u model.User
	err := r.db.QueryRow(ctx, q, login).Scan(&u.ID, &u.Login, &u.PasswordHash, &u.Role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *PgUserRepo) GetByID(ctx context.Context, id int64) (*model.User, error) {
	const q = `
		SELECT id, login, password_hash, role
		FROM users
		WHERE id = $1`

	var u model.User
	err := r.db.QueryRow(ctx, q, id).Scan(&u.ID, &u.Login, &u.PasswordHash, &u.Role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}
