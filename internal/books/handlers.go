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

// UpdateBookProgress atualiza a página atual e publica o evento ReadingProgressUpdated.
//
// @Summary Atualizar progresso
// @Description Atualiza a página atual (o status muda sozinho: reading ou read) e publica o evento ReadingProgressUpdated no NATS.
// @Tags books
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID do livro (UUID)" format(uuid)
// @Param body body updateBookItem true "Nova página atual"
// @Success 200 {object} repo.Book
// @Failure 400 {object} map[string]string "JSON inválido, ID inválido ou página fora do intervalo"
// @Failure 401 {object} map[string]string "Token ausente, inválido ou vencido"
// @Failure 404 {object} map[string]string "Livro não encontrado"
// @Failure 500 {object} map[string]string
// @Router /books/{id}/progress [patch]
func (h *BookHandler) UpdateBookProgress(w http.ResponseWriter, r *http.Request) {
	var tempBook updateBookItem
	if err := json.Read(r, &tempBook); err != nil {
		json.WriteError(w, http.StatusBadRequest, messages.ErrMissingFields.Error())
		return
	}

	userID, ok := userctx.FromContext(r.Context())
	if !ok {
		slog.Error("Invalid User", "Error:", messages.ErrValidUser.Error(), "user_id", userID.String())
		json.WriteError(w, http.StatusInternalServerError, messages.ErrInternalServerError.Error())
		return
	}

	var bookID pgtype.UUID
	tempBookID := chi.URLParam(r, "id")

	if err := bookID.Scan(tempBookID); err != nil {
		json.WriteError(w, http.StatusBadRequest, messages.ErrValidBook.Error())
		return
	}

	book, err := h.service.UpdateBookProgress(r.Context(), userID, bookID, tempBook)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Info("Book not found")
			json.WriteError(w, http.StatusNotFound, messages.ErrPostgressNoRows.Error())
			return
		}
		if errors.Is(err, messages.ErrInvalidPage) {
			slog.Info("Pagina inexistente")
			json.WriteError(w, http.StatusBadRequest, messages.ErrInvalidPage.Error())
			return
		}
		slog.Error("Update book status failed", "error", err.Error(), "book_id", bookID, "user_id", userID)
		json.WriteError(w, http.StatusInternalServerError, messages.ErrInternalServerError.Error())
		return
	}
	json.Write(w, http.StatusOK, book)
}

// GetBookByStatus lista os livros do usuário, com filtro opcional por status.
//
// @Summary Listar livros
// @Description Lista os livros do usuário do token. Sem o parâmetro status, devolve todos.
// @Tags books
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filtrar por status" Enums(wanna_read, reading, read)
// @Success 200 {array} repo.Book
// @Failure 400 {object} map[string]string "Status inválido"
// @Failure 401 {object} map[string]string "Token ausente, inválido ou vencido"
// @Failure 500 {object} map[string]string
// @Router /books [get]
func (h *BookHandler) GetBookByStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := userctx.FromContext(r.Context())
	if !ok {
		slog.Error("Invalid User", "Error:", messages.ErrValidUser.Error(), "user_id", userID.String())
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

// GetBookByID devolve um livro do usuário.
//
// @Summary Detalhe do livro
// @Description Devolve um livro do usuário do token. Livro de outro usuário responde 404.
// @Tags books
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID do livro (UUID)" format(uuid)
// @Success 200 {object} repo.Book
// @Failure 400 {object} map[string]string "ID inválido"
// @Failure 401 {object} map[string]string "Token ausente, inválido ou vencido"
// @Failure 404 {object} map[string]string "Livro não encontrado"
// @Failure 500 {object} map[string]string
// @Router /books/{id} [get]
func (h *BookHandler) GetBookByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := userctx.FromContext(r.Context())
	if !ok {
		slog.Error("Invalid User", "Error:", messages.ErrValidUser.Error(), "user_id", userID.String())
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

// CreateBook cadastra um livro na biblioteca do usuário.
//
// @Summary Cadastrar livro
// @Description Cadastra um livro na biblioteca pessoal do usuário do token. isbn, rating, notes e cover_url são opcionais.
// @Tags books
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body bookItems true "Dados do livro"
// @Success 201 {object} repo.Book
// @Failure 400 {object} map[string]string "JSON inválido ou campo inválido (título, autor, páginas, nota)"
// @Failure 401 {object} map[string]string "Token ausente, inválido ou vencido"
// @Failure 409 {object} map[string]string "ISBN já cadastrado para este usuário"
// @Failure 500 {object} map[string]string
// @Router /books [post]
func (h *BookHandler) CreateBook(w http.ResponseWriter, r *http.Request) {
	var tempBook bookItems
	if err := json.Read(r, &tempBook); err != nil {
		json.WriteError(w, http.StatusBadRequest, messages.ErrMissingFields.Error())
		return
	}

	userID, ok := userctx.FromContext(r.Context())
	if !ok {
		slog.Error("Invalid User", "Error:", messages.ErrValidUser.Error(), "user_id", userID.String())
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
