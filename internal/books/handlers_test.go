package books

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	repo "my-personal-lib/internal/adapters/postgresql/books/sqlc"
	messages "my-personal-lib/internal/error"
	"my-personal-lib/internal/userctx"
)

// fakeService implementa a interface BooksServices sem banco de dados.
// Cada teste diz o que o UpdateBookProgress deve devolver (book e err)
// e, depois da requisição, confere se ele foi chamado e com quais valores.
type fakeService struct {
	book repo.Book
	err  error

	called     bool
	gotUserID  pgtype.UUID
	gotBookID  pgtype.UUID
	gotPayload updateBookItem
}

func (f *fakeService) UpdateBookProgress(ctx context.Context, userID pgtype.UUID, bookID pgtype.UUID, bookItems updateBookItem) (repo.Book, error) {
	f.called = true
	f.gotUserID = userID
	f.gotBookID = bookID
	f.gotPayload = bookItems
	return f.book, f.err
}

// Os outros métodos só existem para satisfazer a interface; estes testes não os usam.
func (f *fakeService) CreateBook(ctx context.Context, userID pgtype.UUID, bookItems bookItems) (repo.Book, error) {
	return repo.Book{}, nil
}

func (f *fakeService) GetBookByID(ctx context.Context, userID pgtype.UUID, bookID pgtype.UUID) (repo.Book, error) {
	return repo.Book{}, nil
}

func (f *fakeService) GetBookByStatus(ctx context.Context, userID pgtype.UUID, status pgtype.Text) ([]repo.Book, error) {
	return nil, nil
}

// newTestRouter monta o mesmo caminho do api.go (middleware + rota),
// para o teste cobrir também o token (Authorization) e o {id} da URL.
func newTestRouter(svc BooksServices) http.Handler {
	h := NewHandler(svc)
	r := chi.NewRouter()
	r.Route("/books", func(r chi.Router) {
		r.Use(userctx.Middleware(testSecret))
		r.Patch("/{id}/progress", h.UpdateBookProgress)
	})
	return r
}

func TestUpdateBookProgress(t *testing.T) {
	const (
		userID = "11111111-1111-1111-1111-111111111111"
		bookID = "22222222-2222-2222-2222-222222222222"
	)

	tests := []struct {
		name string

		// requisição
		path   string
		userID string // vazio = sem o cabeçalho Authorization (sem login)
		body   string

		// o que o serviço falso devolve
		svcBook repo.Book
		svcErr  error

		// o que esperamos
		wantStatus     int
		wantError      string // mensagem do campo "error"; vazio = resposta de sucesso
		wantSvcCalled  bool
		wantPage       int32  // current_page que o handler deve repassar ao serviço
		wantBodyStatus string // status do livro no corpo da resposta de sucesso
	}{
		{
			name:           "caminho feliz devolve 200 com o livro",
			path:           "/books/" + bookID + "/progress",
			userID:         userID,
			body:           `{"current_page": 120}`,
			svcBook:        repo.Book{CurrentPage: 120, TotalPages: 300, Status: "reading"},
			wantStatus:     http.StatusOK,
			wantSvcCalled:  true,
			wantPage:       120,
			wantBodyStatus: "reading",
		},
		{
			name:          "página fora do intervalo devolve 400",
			path:          "/books/" + bookID + "/progress",
			userID:        userID,
			body:          `{"current_page": 301}`,
			svcErr:        messages.ErrInvalidPage,
			wantStatus:    http.StatusBadRequest,
			wantError:     messages.ErrInvalidPage.Error(),
			wantSvcCalled: true,
			wantPage:      301,
		},
		{
			name:          "livro inexistente ou de outro usuário devolve 404",
			path:          "/books/" + bookID + "/progress",
			userID:        userID,
			body:          `{"current_page": 10}`,
			svcErr:        pgx.ErrNoRows,
			wantStatus:    http.StatusNotFound,
			wantError:     messages.ErrPostgressNoRows.Error(),
			wantSvcCalled: true,
			wantPage:      10,
		},
		{
			name:          "erro inesperado devolve 500 sem vazar a mensagem interna",
			path:          "/books/" + bookID + "/progress",
			userID:        userID,
			body:          `{"current_page": 10}`,
			svcErr:        errors.New("connection refused"),
			wantStatus:    http.StatusInternalServerError,
			wantError:     messages.ErrInternalServerError.Error(),
			wantSvcCalled: true,
			wantPage:      10,
		},
		{
			name:          "corpo com status é recusado (DisallowUnknownFields)",
			path:          "/books/" + bookID + "/progress",
			userID:        userID,
			body:          `{"current_page": 10, "status": "read"}`,
			wantStatus:    http.StatusBadRequest,
			wantError:     messages.ErrMissingFields.Error(),
			wantSvcCalled: false,
		},
		{
			name:          "JSON quebrado devolve 400",
			path:          "/books/" + bookID + "/progress",
			userID:        userID,
			body:          `{"current_page": `,
			wantStatus:    http.StatusBadRequest,
			wantError:     messages.ErrMissingFields.Error(),
			wantSvcCalled: false,
		},
		{
			name:          "id do livro que não é UUID devolve 400",
			path:          "/books/abc/progress",
			userID:        userID,
			body:          `{"current_page": 10}`,
			wantStatus:    http.StatusBadRequest,
			wantError:     messages.ErrValidBook.Error(),
			wantSvcCalled: false,
		},
		{
			name:          "sem token (não logado) devolve 401",
			path:          "/books/" + bookID + "/progress",
			userID:        "",
			body:          `{"current_page": 10}`,
			wantStatus:    http.StatusUnauthorized,
			wantError:     messages.ErrMissingToken.Error(),
			wantSvcCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeService{book: tt.svcBook, err: tt.svcErr}

			// httptest cria a requisição e grava a resposta em memória, sem abrir porta.
			req := httptest.NewRequest(http.MethodPatch, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.userID != "" {
				req.Header.Set("Authorization", bearer(t, tt.userID))
			}
			rec := httptest.NewRecorder()

			newTestRouter(svc).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, quero %d (corpo: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}

			if svc.called != tt.wantSvcCalled {
				t.Fatalf("serviço chamado = %v, quero %v", svc.called, tt.wantSvcCalled)
			}

			// Quando o serviço é chamado, o handler deve repassar os valores certos.
			if tt.wantSvcCalled {
				if svc.gotPayload.CurrentPage != tt.wantPage {
					t.Errorf("current_page repassado = %d, quero %d", svc.gotPayload.CurrentPage, tt.wantPage)
				}
				if got := svc.gotUserID.String(); got != userID {
					t.Errorf("user_id repassado = %s, quero %s", got, userID)
				}
				if got := svc.gotBookID.String(); got != bookID {
					t.Errorf("book_id repassado = %s, quero %s", got, bookID)
				}
			}

			if tt.wantError != "" {
				var body struct {
					Error string `json:"error"`
				}
				if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
					t.Fatalf("corpo de erro não é JSON válido: %v", err)
				}
				if body.Error != tt.wantError {
					t.Errorf("error = %q, quero %q", body.Error, tt.wantError)
				}
				return
			}

			var book repo.Book
			if err := json.NewDecoder(rec.Body).Decode(&book); err != nil {
				t.Fatalf("corpo de sucesso não é um livro em JSON: %v", err)
			}
			if book.Status != tt.wantBodyStatus {
				t.Errorf("status do livro = %q, quero %q", book.Status, tt.wantBodyStatus)
			}
		})
	}
}
