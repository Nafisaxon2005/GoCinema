package jwtutil

import (
	"testing"
	"time"

	"github.com/raxima/seatpicker/internal/model"
)

func TestGenerateAndParseAccess(t *testing.T) {
	secret := []byte("test-secret")

	t.Run("валидный токен парсится корректно", func(t *testing.T) {
		token, err := GenerateAccess(42, model.RoleAdmin, secret, 15*time.Minute)
		if err != nil {
			t.Fatalf("GenerateAccess вернул ошибку: %v", err)
		}
		if token == "" {
			t.Fatal("токен не должен быть пустым")
		}

		claims, err := ParseAccess(token, secret)
		if err != nil {
			t.Fatalf("ParseAccess вернул ошибку: %v", err)
		}
		if claims.UserID != 42 {
			t.Errorf("UserID = %d, ожидалось 42", claims.UserID)
		}
		if claims.Role != model.RoleAdmin {
			t.Errorf("Role = %q, ожидалось %q", claims.Role, model.RoleAdmin)
		}
	})

	t.Run("истёкший токен не проходит проверку", func(t *testing.T) {
		token, err := GenerateAccess(1, model.RoleViewer, secret, -1*time.Minute)
		if err != nil {
			t.Fatalf("GenerateAccess вернул ошибку: %v", err)
		}

		_, err = ParseAccess(token, secret)
		if err != ErrInvalidToken {
			t.Errorf("ожидалась ошибка %v, получено %v", ErrInvalidToken, err)
		}
	})

	t.Run("неверный секрет не проходит проверку", func(t *testing.T) {
		token, err := GenerateAccess(1, model.RoleViewer, secret, 15*time.Minute)
		if err != nil {
			t.Fatalf("GenerateAccess вернул ошибку: %v", err)
		}

		_, err = ParseAccess(token, []byte("wrong-secret"))
		if err != ErrInvalidToken {
			t.Errorf("ожидалась ошибка %v, получено %v", ErrInvalidToken, err)
		}
	})

	t.Run("мусорная строка не парсится", func(t *testing.T) {
		_, err := ParseAccess("not-a-jwt-token", secret)
		if err != ErrInvalidToken {
			t.Errorf("ожидалась ошибка %v, получено %v", ErrInvalidToken, err)
		}
	})
}

func TestNewRefreshToken(t *testing.T) {
	raw1, hash1, err := NewRefreshToken()
	if err != nil {
		t.Fatalf("NewRefreshToken вернул ошибку: %v", err)
	}
	if raw1 == "" || hash1 == "" {
		t.Fatal("raw и hash не должны быть пустыми")
	}

	raw2, hash2, err := NewRefreshToken()
	if err != nil {
		t.Fatalf("NewRefreshToken вернул ошибку: %v", err)
	}

	if raw1 == raw2 {
		t.Error("два вызова подряд не должны давать одинаковый raw-токен")
	}
	if hash1 == hash2 {
		t.Error("два вызова подряд не должны давать одинаковый hash")
	}

	t.Run("хэш детерминирован для одного и того же raw", func(t *testing.T) {
		if HashRefreshToken(raw1) != hash1 {
			t.Error("HashRefreshToken должен давать одинаковый результат для одного raw")
		}
	})

	t.Run("разные raw дают разные хэши", func(t *testing.T) {
		if HashRefreshToken(raw1) == HashRefreshToken(raw2) {
			t.Error("разные raw-токены не должны давать одинаковый хэш")
		}
	})
}
