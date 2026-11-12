package books

import (
	"errors"
	"log/slog"
	"strings"

	messages "my-personal-lib/internal/error"
	"my-personal-lib/internal/json"
	"my-personal-lib/internal/userctx"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type BookHandler struct {
	service BooksServices
}

func NewHandler(service BooksServices) *BookHandler {
	return &BookHandler{
		service: service,
	}
}

func (h *BookHandler) GetBookByStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := userctx.FromContext(r.Context())
	if !ok {
		slog.Error("Invalid user", "user_id", userID.String())
		json.WriteError(w, http.StatusInternalServerError, messages.ErrInternalServerError.Error())
		return
	}
	var status pgtype.Text
	tempStatus := r.URL.Query().Get("status")
	tempStatus = strings.TrimSpace(tempStatus)
	if tempStatus != "" {
		status.Scan(tempStatus)
	}

	book, err := h.service.GetBookByStatus(r.Context(), userID, status)
	if err != nil {
		if errors.Is(err, messages.ErrInvalidStatus) {
			json.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		slog.Error("Get book by status failed", "error", err.Error(), "status", status.String)
		json.WriteError(w, http.StatusInternalServerError, messages.ErrInternalServerError.Error())
		return
	}
	json.Write(w, http.StatusOK, book)
}

func (h *BookHandler) GetBookByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := userctx.FromContext(r.Context())
	if !ok {
		slog.Error("Invalid user", "user_id", userID.String())
		json.WriteError(w, http.StatusInternalServerError, messages.ErrInternalServerError.Error())
		return
	}
	var bookID pgtype.UUID
	tempBookID := chi.URLParam(r, "id")

	if err := bookID.Scan(tempBookID); err != nil {
		json.WriteError(w, http.StatusBadRequest, messages.ErrValidBook.Error())
		return
	}

	book, err := h.service.GetBookByID(r.Context(), userID, bookID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Info("Book not found")
			json.WriteError(w, http.StatusNotFound, messages.ErrPostgressNoRows.Error())
			return
		}
		slog.Error("Get book failed", "error", err.Error(), "book_id", bookID.String())
		json.WriteError(w, http.StatusInternalServerError, messages.ErrInternalServerError.Error())
		return
	}
	json.Write(w, http.StatusOK, book)
}

func (h *BookHandler) CreateBook(w http.ResponseWriter, r *http.Request) {
	var tempBook bookItems
	if err := json.Read(r, &tempBook); err != nil {
		json.WriteError(w, http.StatusBadRequest, messages.ErrMissingFields.Error())
		return
	}

	userID, ok := userctx.FromContext(r.Context())
	if !ok {
		slog.Error(messages.ErrValidUser.Error())
		json.WriteError(w, http.StatusInternalServerError, messages.ErrInternalServerError.Error())
		return
	}

	createdBook, err := h.service.CreateBook(r.Context(), userID, tempBook)
	if err != nil {
		var pg *pgconn.PgError
		switch {
		case errors.Is(err, messages.ErrEmptyAuthor) || errors.Is(err, messages.ErrEmptyTitle) || errors.Is(err, messages.ErrPageTotal) || errors.Is(err, messages.ErrRating) || errors.Is(err, messages.ErrValidUser):
			json.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.As(err, &pg) && pg.Code == "23505":
			json.WriteError(w, http.StatusConflict, messages.ErrISBN.Error())
		default:
			json.WriteError(w, http.StatusInternalServerError, messages.ErrInternalServerError.Error())
			slog.Error("Create book failed", "error", err.Error(), "user_id", userID.String())
		}

		return
	}

	json.Write(w, http.StatusCreated, createdBook)
}
