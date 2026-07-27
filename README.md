# SeatPicker

Сервис бронирования мест на сеансы.

## Структура (J-01)

```
cmd/server        — точка входа (main.go)
internal/router    — сборка gin.Engine и маршрутов
internal/handler    — HTTP-хендлеры
internal/service    — бизнес-логика
internal/repository — доступ к БД (интерфейсы + pgx-реализации)
internal/middleware  — JWT-аутентификация, проверка ролей
internal/model      — сущности, статусы, sentinel-ошибки
migrations         — SQL DDL
```

## Запуск

```bash
docker compose up --build
```

Проверка: `GET http://localhost:8080/health` → `{"status":"ok"}`

## Контракт статусов (J-02)

- `shows`: `draft -> published -> cancelled`
- `seats`: `free -> booked`
- `bookings`: `booked -> cancelled`

## Sentinel-ошибки

`ErrNotFound`, `ErrInvalid`, `ErrForbidden`, `ErrSeatTaken`, `ErrClosed` — в `internal/model/errors.go`.
Хендлеры мапят их на HTTP-коды: 404 / 400 / 403 / 409 / 409.
