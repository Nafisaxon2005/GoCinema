package model

import "errors"

// Sentinel-ошибки — единый контракт на троих (J-02).
// handler мапит их на HTTP-коды: NotFound->404, Invalid->400, Forbidden->403,
// SeatTaken->409, Closed->409.
var (
	ErrNotFound  = errors.New("resource not found")
	ErrInvalid   = errors.New("invalid input")
	ErrForbidden = errors.New("forbidden")
	ErrSeatTaken = errors.New("seat already taken")
	ErrClosed    = errors.New("show is closed")
)
