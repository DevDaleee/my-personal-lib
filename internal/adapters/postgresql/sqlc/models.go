package repo

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

const (
	StatusWantToRead = "wanna_read"
	StatusReading    = "reading"
	StatusRead       = "read"
)

type Book struct {
	ID          pgtype.UUID `json:"id"`
	UserID      pgtype.UUID `json:"user_id"`
	Title       string      `json:"title"`
	Author      string      `json:"author"`
	Isbn        *string     `json:"isbn"`
	TotalPages  int         `json:"total_pages"`
	CurrentPage int         `json:"current_page"`
	Status      string      `json:"status"`
	Rating      *int        `json:"rating"`
	Notes       *string     `json:"notes"`
	CoverURL    *string     `json:"cover_url"`
	Owned       bool        `json:"owned"`
	StartedAt   *time.Time  `json:"started_at"`
	FinishedAt  *time.Time  `json:"finished_at"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type ProgressUpdate struct {
	ID           pgtype.UUID `json:"id"`
	BookID       pgtype.UUID `json:"book_id"`
	PreviousPage int         `json:"previous_page"`
	CurrentPage  int         `json:"current_page"`
	Status       string      `json:"status"`
	CreatedAt    time.Time   `json:"created_at"`
}
