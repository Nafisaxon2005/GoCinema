package model

import "errors"

var (
	ErrNotFound      = errors.New("resource not found")
	ErrInvalid       = errors.New("invalid input")
	ErrForbidden     = errors.New("forbidden")
	ErrSeatTaken     = errors.New("seat already taken")
	ErrClosed        = errors.New("show is closed")
	ErrAlreadyExists = errors.New("already exists")
)
