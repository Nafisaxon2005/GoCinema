package seats

import "errors"

var (
	ErrSeatTaken        = errors.New("seat already taken")
	ErrShowNotAvailable = errors.New("show is cancelled or not published")
)
