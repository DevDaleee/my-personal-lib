package books

import "github.com/jackc/pgx/v5/pgtype"

type bookItems struct {
	Title      string      `json:"title"`
	Author     string      `json:"author"`
	Isbn       pgtype.Text `json:"isbn"`
	TotalPages int32       `json:"total_pages"`
	Rating     pgtype.Int4 `json:"rating"`
	Notes      pgtype.Text `json:"notes"`
	CoverUrl   pgtype.Text `json:"cover_url"`
	Owned      bool        `json:"owned"`
}
