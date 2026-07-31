package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"

	"github.com/raxima/seatpicker/internal/model"
)

func TestPgRefreshTokenRepo_Create(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("не удалось создать pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewPgRefreshTokenRepo(mock)
	expiresAt := time.Now().Add(24 * time.Hour)

	mock.ExpectQuery(`INSERT INTO refresh_tokens`).
		WithArgs(int64(1), "hash123", expiresAt).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(10)))

	id, err := repo.Create(context.Background(), &model.RefreshToken{
		UserID: 1, TokenHash: "hash123", ExpiresAt: expiresAt,
	})
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if id != 10 {
		t.Errorf("id = %d, ожидалось 10", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("не все ожидания выполнены: %v", err)
	}
}

func TestPgRefreshTokenRepo_GetByHash(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("не удалось создать pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewPgRefreshTokenRepo(mock)
	now := time.Now()

	t.Run("токен найден и активен", func(t *testing.T) {
		mock.ExpectQuery(`SELECT id, user_id, token_hash, expires_at, revoked_at, created_at`).
			WithArgs("hash-active").
			WillReturnRows(pgxmock.NewRows([]string{"id", "user_id", "token_hash", "expires_at", "revoked_at", "created_at"}).
				AddRow(int64(1), int64(2), "hash-active", now.Add(time.Hour), nil, now))

		rt, err := repo.GetByHash(context.Background(), "hash-active")
		if err != nil {
			t.Fatalf("неожиданная ошибка: %v", err)
		}
		if rt.ID != 1 || rt.UserID != 2 {
			t.Errorf("неверные данные токена: %+v", rt)
		}
		if rt.RevokedAt != nil {
			t.Error("RevokedAt должен быть nil для активного токена")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("не все ожидания выполнены: %v", err)
		}
	})

	t.Run("токен не найден -> ErrNotFound", func(t *testing.T) {
		mock.ExpectQuery(`SELECT id, user_id, token_hash, expires_at, revoked_at, created_at`).
			WithArgs("hash-missing").
			WillReturnError(pgx.ErrNoRows)

		_, err := repo.GetByHash(context.Background(), "hash-missing")
		if !errors.Is(err, model.ErrNotFound) {
			t.Errorf("ожидалась ошибка %v, получено %v", model.ErrNotFound, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("не все ожидания выполнены: %v", err)
		}
	})
}

func TestPgRefreshTokenRepo_Revoke(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("не удалось создать pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewPgRefreshTokenRepo(mock)

	t.Run("успешный отзыв", func(t *testing.T) {
		mock.ExpectExec(`UPDATE refresh_tokens`).
			WithArgs(int64(5), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		if err := repo.Revoke(context.Background(), 5); err != nil {
			t.Fatalf("неожиданная ошибка: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("не все ожидания выполнены: %v", err)
		}
	})

	t.Run("ошибка БД пробрасывается", func(t *testing.T) {
		dbErr := errors.New("db down")
		mock.ExpectExec(`UPDATE refresh_tokens`).
			WithArgs(int64(6), pgxmock.AnyArg()).
			WillReturnError(dbErr)

		err := repo.Revoke(context.Background(), 6)
		if err == nil {
			t.Fatal("ожидалась ошибка")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("не все ожидания выполнены: %v", err)
		}
	})
}

func TestPgRefreshTokenRepo_RevokeAllForUser(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("не удалось создать pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewPgRefreshTokenRepo(mock)

	mock.ExpectExec(`UPDATE refresh_tokens`).
		WithArgs(int64(3), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 2))

	if err := repo.RevokeAllForUser(context.Background(), 3); err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("не все ожидания выполнены: %v", err)
	}
}
