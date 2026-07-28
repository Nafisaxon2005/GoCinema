// Package jwtutil отвечает за генерацию и проверку access-токенов (JWT)
// и генерацию/хэширование refresh-токенов. Не знает ни про gin, ни про БД —
// используется и сервисом (выдача токенов), и middleware (проверка).
package jwtutil

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/raxima/seatpicker/internal/model"
)

var ErrInvalidToken = errors.New("invalid or expired token")

// Claims — то, что лежит внутри access-токена.
type Claims struct {
	jwt.RegisteredClaims
	UserID int64      `json:"uid"`
	Role   model.Role `json:"role"`
}

// GenerateAccess создаёт короткоживущий access-токен (~15 мин), stateless —
// в БД не сохраняется, отзыв не поддерживается (для этого есть refresh).
func GenerateAccess(userID int64, role model.Role, secret []byte, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
		UserID: userID,
		Role:   role,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// ParseAccess проверяет подпись и срок действия access-токена и возвращает claims.
func ParseAccess(tokenStr string, secret []byte) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		// Явно запрещаем алгоритмы кроме HMAC — защита от "alg confusion".
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return secret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// NewRefreshToken генерирует случайный refresh-токен (raw — отдаётся клиенту)
// и его sha256-хэш (hash — хранится в БД, raw в БД никогда не попадает).
func NewRefreshToken() (raw string, hash string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	raw = base64.RawURLEncoding.EncodeToString(buf)
	return raw, HashRefreshToken(raw), nil
}

// HashRefreshToken считает sha256-хэш сырого refresh-токена — используется
// и при выдаче (что сохранить), и при refresh/logout (что искать в БД).
func HashRefreshToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
