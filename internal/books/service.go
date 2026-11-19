package books

import (
	"context"
	repo "my-personal-lib/internal/adapters/postgresql/books/sqlc"
	messages "my-personal-lib/internal/error"
	"my-personal-lib/internal/events"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go/jetstream"
)

type BooksServices interface {
	CreateBook(ctx context.Context, userID pgtype.UUID, bookItems bookItems) (repo.Book, error)
	GetBookByID(ctx context.Context, userID pgtype.UUID, bookID pgtype.UUID) (repo.Book, error)
	GetBookByStatus(ctx context.Context, userID pgtype.UUID, status pgtype.Text) ([]repo.Book, error)
	UpdateBookProgress(ctx context.Context, userID pgtype.UUID, bookID pgtype.UUID, bookItems updateBookItem) (repo.Book, error)
}

type svc struct {
	repo *repo.Queries
	db   *pgxpool.Pool
	js   jetstream.JetStream
}

func NewService(repo *repo.Queries, db *pgxpool.Pool, js jetstream.JetStream) BooksServices {
	return &svc{repo: repo, db: db, js: js}
}

func (s *svc) UpdateBookProgress(ctx context.Context, userID pgtype.UUID, bookID pgtype.UUID, bookItems updateBookItem) (repo.Book, error) {
	if !userID.Valid {
		return repo.Book{}, messages.ErrValidUser
	}
	if !bookID.Valid {
		return repo.Book{}, messages.ErrValidBook
	}

	if bookItems.CurrentPage < 0 {
		return repo.Book{}, messages.ErrInvalidPage
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return repo.Book{}, err
	}
	defer tx.Rollback(ctx)
	qtx := s.repo.WithTx(tx)

	book, err := qtx.GetBookForUpdate(ctx, repo.GetBookForUpdateParams{
		UserID: userID,
		ID:     bookID,
	})
	if err != nil {
		return repo.Book{}, err
	}

	if bookItems.CurrentPage > book.TotalPages {
		return repo.Book{}, messages.ErrInvalidPage
	}

	bookStatus := statusForPage(bookItems.CurrentPage, book.TotalPages)

	progress, err := qtx.CreateProgressUpdate(ctx, repo.CreateProgressUpdateParams{
		BookID:       bookID,
		PreviousPage: book.CurrentPage,
		Status:       bookStatus,
		CurrentPage:  bookItems.CurrentPage,
	})
	if err != nil {
		return repo.Book{}, err
	}

	updatedBook, err := qtx.UpdateProgress(ctx, repo.UpdateProgressParams{
		UserID:      userID,
		ID:          bookID,
		Status:      bookStatus,
		CurrentPage: bookItems.CurrentPage,
	})

	evt := events.ReadingProgressUpdated{
		EventID:      progress.ID,
		Event:        events.EventProgressUpdated,
		BookID:       bookID,
		UserID:       userID,
		PreviousPage: book.CurrentPage,
		CurrentPage:  updatedBook.CurrentPage,
		TotalPages:   book.TotalPages,
		Status:       updatedBook.Status,
		UpdatedAt:    progress.CreatedAt.Time,
	}

	if err != nil {
		return repo.Book{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return repo.Book{}, err
	}
	s.publishProgressUpdated(ctx, evt)
	return updatedBook, nil
}

func (s *svc) GetBookByStatus(ctx context.Context, userID pgtype.UUID, status pgtype.Text) ([]repo.Book, error) {
	if !userID.Valid {
		return []repo.Book{}, messages.ErrValidUser
	}
	status.String = strings.TrimSpace(status.String)
	if status.Valid && !(status.String == reading || status.String == wanna_read || status.String == read) {
		return []repo.Book{}, messages.ErrInvalidStatus
	}
	book, err := s.repo.ListBooks(ctx, repo.ListBooksParams{
		UserID: userID,
		Status: status,
	})
	if err != nil {
		return []repo.Book{}, err
	}
	return book, nil
}

func (s *svc) GetBookByID(ctx context.Context, userID pgtype.UUID, bookID pgtype.UUID) (repo.Book, error) {
	if !userID.Valid {
		return repo.Book{}, messages.ErrValidUser
	}
	if !bookID.Valid {
		return repo.Book{}, messages.ErrValidBook
	}

	book, err := s.repo.GetBook(ctx, repo.GetBookParams{
		ID:     bookID,
		UserID: userID,
	})
	if err != nil {
		return repo.Book{}, err
	}

	return book, nil
}

func (s *svc) CreateBook(ctx context.Context, userID pgtype.UUID, bookData bookItems) (repo.Book, error) {
	if !userID.Valid {
		return repo.Book{}, messages.ErrValidUser
	}

	bookData.Author = strings.TrimSpace(bookData.Author)
	bookData.Title = strings.TrimSpace(bookData.Title)

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
