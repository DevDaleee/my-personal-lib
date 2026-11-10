package books

import (
	"context"
	repo "my-personal-lib/internal/adapters/postgresql/sqlc"
	messages "my-personal-lib/internal/error"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BooksServices interface {
	CreateBook(ctx context.Context, userID pgtype.UUID, bookItems bookItems) (repo.Book, error)
}

type svc struct {
	repo *repo.Queries
	db   *pgxpool.Pool
}

func NewSerice(repo *repo.Queries, db *pgxpool.Pool) BooksServices {
	return &svc{repo: repo, db: db}
}

func (s *svc) CreateBook(ctx context.Context, userID pgtype.UUID, bookData bookItems) (repo.Book, error) {
	if !userID.Valid {
		return repo.Book{}, messages.ErrValidUser
	}
	if bookData.Author == "" {
		return repo.Book{}, messages.ErrEmptyAuthor
	}
	if bookData.Title == "" {
		return repo.Book{}, messages.ErrEmptyTitle
	}
	if bookData.TotalPages <= 0 {
		return repo.Book{}, messages.ErrPageTotal
	}
	if (bookData.Rating.Int32 <= 0 || bookData.Rating.Int32 > 5) && bookData.Rating.Valid {
		return repo.Book{}, messages.ErrRating
	}

	if bookData.Isbn.Valid && bookData.Isbn.String == "" {
		bookData.Isbn.Valid = false
	}
	if bookData.CoverUrl.Valid && bookData.CoverUrl.String == "" {
		bookData.CoverUrl.Valid = false
	}
	if bookData.Notes.Valid && bookData.Notes.String == "" {
		bookData.Notes.Valid = false
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return repo.Book{}, err
	}
	defer tx.Rollback(ctx)
	qtx := s.repo.WithTx(tx)

	book, err := qtx.CreateBook(ctx, repo.CreateBookParams{
		UserID:     userID,
		Title:      bookData.Title,
		Author:     bookData.Author,
		TotalPages: bookData.TotalPages,
		CoverUrl:   bookData.CoverUrl,
		Isbn:       bookData.Isbn,
		Owned:      bookData.Owned,
		Rating:     bookData.Rating,
		Notes:      bookData.Notes,
	})
	if err != nil {
		return repo.Book{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return repo.Book{}, err
	}

	return book, nil
}
