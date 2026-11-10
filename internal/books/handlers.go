package books

import (
	"errors"
	"log/slog"

	messages "my-personal-lib/internal/error"
	"my-personal-lib/internal/json"
	"my-personal-lib/internal/userctx"
	"net/http"

	"github.com/jackc/pgx/v5/pgconn"
)

type BookHandler struct {
	service BooksServices
}

func NewHandler(service BooksServices) *BookHandler {
	return &BookHandler{
		service: service,
	}
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
