package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/raxima/seatpicker/internal/model"
)

type PgRefreshTokenRepo struct {
	db *pgxpool.Pool
}

func NewPgRefreshTokenRepo(db *pgxpool.Pool) *PgRefreshTokenRepo {
	return &PgRefreshTokenRepo{db: db}
}

func (r *PgRefreshTokenRepo) Create(ctx context.Context, rt *model.RefreshToken) (int64, error) {
	const q = `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id`

	var id int64
	err := r.db.QueryRow(ctx, q, rt.UserID, rt.TokenHash, rt.ExpiresAt).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *PgRefreshTokenRepo) GetByHash(ctx context.Context, hash string) (*model.RefreshToken, error) {
	const q = `
		SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
		FROM refresh_tokens
		WHERE token_hash = $1`

	var rt model.RefreshToken
	err := r.db.QueryRow(ctx, q, hash).Scan(
		&rt.ID, &rt.UserID, &rt.TokenHash, &rt.ExpiresAt, &rt.RevokedAt, &rt.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	return &rt, nil
}

func (r *PgRefreshTokenRepo) Revoke(ctx context.Context, id int64) error {
	const q = `
		UPDATE refresh_tokens
		SET revoked_at = $2
		WHERE id = $1 AND revoked_at IS NULL`

	_, err := r.db.Exec(ctx, q, id, time.Now())
	return err
}

func (r *PgRefreshTokenRepo) RevokeAllForUser(ctx context.Context, userID int64) error {
	const q = `
		UPDATE refresh_tokens
		SET revoked_at = $2
		WHERE user_id = $1 AND revoked_at IS NULL`

	_, err := r.db.Exec(ctx, q, userID, time.Now())
	return err
}
