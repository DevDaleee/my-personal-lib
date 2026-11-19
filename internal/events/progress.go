package events

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

const (
	StreamName             = "BOOKS"
	SubjectProgressUpdated = "books.progress.updated"
	EventProgressUpdated   = "ReadingProgressUpdated"
)

type ReadingProgressUpdated struct {
	EventID      pgtype.UUID `json:"event_id"`
	Event        string      `json:"event"`
	BookID       pgtype.UUID `json:"book_id"`
	UserID       pgtype.UUID `json:"user_id"`
	PreviousPage int32       `json:"previous_page"`
	CurrentPage  int32       `json:"current_page"`
	TotalPages   int32       `json:"total_pages"`
	Status       string      `json:"status"`
	UpdatedAt    time.Time   `json:"updated_at"`
}
