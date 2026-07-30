package model

import "time"

// Роли пользователя.
type Role string

const (
	RoleViewer    Role = "viewer"
	RoleOrganizer Role = "organizer"
	RoleAdmin     Role = "admin"
)

// Статусы сеанса: draft -> published -> cancelled.
type ShowStatus string

const (
	ShowDraft     ShowStatus = "draft"
	ShowPublished ShowStatus = "published"
	ShowCancelled ShowStatus = "cancelled"
)

// Статусы места: free -> booked.
type SeatStatus string

const (
	SeatFree   SeatStatus = "free"
	SeatBooked SeatStatus = "booked"
)

// Статусы брони: booked -> cancelled.
type BookingStatus string

const (
	BookingBooked    BookingStatus = "booked"
	BookingCancelled BookingStatus = "cancelled"
)

type User struct {
	ID           int64  `json:"id" db:"id"`
	Login        string `json:"login" db:"login"`
	PasswordHash string `json:"-" db:"password_hash"`
	Role         Role   `json:"role" db:"role"`
}

type Show struct {
	ID          int64      `json:"id" db:"id"`
	OrganizerID int64      `json:"organizer_id" db:"organizer_id"`
	Title       string     `json:"title" db:"title"`
	Venue       string     `json:"venue" db:"venue"`
	StartsAt    time.Time  `json:"starts_at" db:"starts_at"`
	Status      ShowStatus `json:"status" db:"status"`
	PosterPath  string     `json:"poster_path,omitempty" db:"poster_path"`
}

type Seat struct {
	ID     int64      `json:"id" db:"id"`
	ShowID int64      `json:"show_id" db:"show_id"`
	Row    int        `json:"row" db:"row"`
	Num    int        `json:"num" db:"num"`
	Price  int64      `json:"price" db:"price"`
	Status SeatStatus `json:"status" db:"status"`
}

type Booking struct {
	ID        int64         `json:"id" db:"id"`
	ShowID    int64         `json:"show_id" db:"show_id"`
	SeatID    int64         `json:"seat_id" db:"seat_id"`
	UserID    int64         `json:"user_id" db:"user_id"`
	Status    BookingStatus `json:"status" db:"status"`
	CreatedAt time.Time     `json:"created_at" db:"created_at"`
}
type BookingFilter struct {
	UserID int64
	Status string
	Date   string
	Limit  int
	Offset int
}

type BookingResponse struct {
	ID        int64  `json:"id"`
	ShowID    int64  `json:"showId"`
	SeatID    int64  `json:"seatId"`
	CreatedAt string `json:"createdAt"`
	ShowDate  string `json:"showDate"`
	Status    string `json:"status"`
}
type ShowDetail struct {
	Show
	FreeSeats int `json:"free_seats"`
}

type ShowListResponse struct {
	Items    []Show `json:"items"`
	Total    int    `json:"total"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}

// RefreshToken — refresh-токен пользователя. Храним только хэш (TokenHash),
// сырой токен никогда не попадает в БД.
type RefreshToken struct {
	ID        int64      `json:"id" db:"id"`
	UserID    int64      `json:"user_id" db:"user_id"`
	TokenHash string     `json:"-" db:"token_hash"`
	ExpiresAt time.Time  `json:"expires_at" db:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}
