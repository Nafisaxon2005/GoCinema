package repository

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"

	"github.com/raxima/seatpicker/internal/model"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

func TestPgUserRepo_Create(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("не удалось создать pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewPgUserRepo(mock, testLogger())

	t.Run("успешное создание", func(t *testing.T) {
		mock.ExpectQuery(`INSERT INTO users`).
			WithArgs("alice", "hash123", model.RoleViewer).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(1)))

		id, err := repo.Create(context.Background(), &model.User{
			Login: "alice", PasswordHash: "hash123", Role: model.RoleViewer,
		})
		if err != nil {
			t.Fatalf("неожиданная ошибка: %v", err)
		}
		if id != 1 {
			t.Errorf("id = %d, ожидалось 1", id)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("не все ожидания выполнены: %v", err)
		}
	})

	t.Run("дублирующийся логин -> ErrAlreadyExists", func(t *testing.T) {
		pgErr := &pgconn.PgError{Code: uniqueViolationCode}

		mock.ExpectQuery(`INSERT INTO users`).
			WithArgs("bob", "hash456", model.RoleViewer).
			WillReturnError(pgErr)

		_, err := repo.Create(context.Background(), &model.User{
			Login: "bob", PasswordHash: "hash456", Role: model.RoleViewer,
		})
		if !errors.Is(err, model.ErrAlreadyExists) {
			t.Errorf("ожидалась ошибка %v, получено %v", model.ErrAlreadyExists, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("не все ожидания выполнены: %v", err)
		}
	})

	t.Run("прочая ошибка БД пробрасывается как есть", func(t *testing.T) {
		dbErr := errors.New("connection lost")

		mock.ExpectQuery(`INSERT INTO users`).
			WithArgs("carol", "hash789", model.RoleViewer).
			WillReturnError(dbErr)

		_, err := repo.Create(context.Background(), &model.User{
			Login: "carol", PasswordHash: "hash789", Role: model.RoleViewer,
		})
		if err == nil {
			t.Fatal("ожидалась ошибка")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("не все ожидания выполнены: %v", err)
		}
	})
}

func TestPgUserRepo_GetByLogin(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("не удалось создать pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewPgUserRepo(mock, testLogger())

	t.Run("пользователь найден", func(t *testing.T) {
		mock.ExpectQuery(`SELECT id, login, password_hash, role`).
			WithArgs("dave").
			WillReturnRows(pgxmock.NewRows([]string{"id", "login", "password_hash", "role"}).
				AddRow(int64(5), "dave", "hashhash", model.RoleOrganizer))

		u, err := repo.GetByLogin(context.Background(), "dave")
		if err != nil {
			t.Fatalf("неожиданная ошибка: %v", err)
		}
		if u.ID != 5 || u.Login != "dave" || u.Role != model.RoleOrganizer {
			t.Errorf("неверные данные пользователя: %+v", u)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("не все ожидания выполнены: %v", err)
		}
	})

	t.Run("пользователь не найден -> ErrNotFound", func(t *testing.T) {
		mock.ExpectQuery(`SELECT id, login, password_hash, role`).
			WithArgs("ghost").
			WillReturnError(pgx.ErrNoRows)

		_, err := repo.GetByLogin(context.Background(), "ghost")
		if !errors.Is(err, model.ErrNotFound) {
			t.Errorf("ожидалась ошибка %v, получено %v", model.ErrNotFound, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("не все ожидания выполнены: %v", err)
		}
	})
}

func TestPgUserRepo_GetByID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("не удалось создать pgxmock: %v", err)
	}
	defer mock.Close()

	repo := NewPgUserRepo(mock, testLogger())

	t.Run("пользователь найден", func(t *testing.T) {
		mock.ExpectQuery(`SELECT id, login, password_hash, role`).
			WithArgs(int64(7)).
			WillReturnRows(pgxmock.NewRows([]string{"id", "login", "password_hash", "role"}).
				AddRow(int64(7), "erin", "hashhash", model.RoleAdmin))

		u, err := repo.GetByID(context.Background(), 7)
		if err != nil {
			t.Fatalf("неожиданная ошибка: %v", err)
		}
		if u.ID != 7 || u.Login != "erin" {
			t.Errorf("неверные данные пользователя: %+v", u)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("не все ожидания выполнены: %v", err)
		}
	})

	t.Run("не найден -> ErrNotFound", func(t *testing.T) {
		mock.ExpectQuery(`SELECT id, login, password_hash, role`).
			WithArgs(int64(999)).
			WillReturnError(pgx.ErrNoRows)

		_, err := repo.GetByID(context.Background(), 999)
		if !errors.Is(err, model.ErrNotFound) {
			t.Errorf("ожидалась ошибка %v, получено %v", model.ErrNotFound, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("не все ожидания выполнены: %v", err)
		}
	})
}
