package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"

	repo "my-personal-lib/internal/adapters/postgresql/books/sqlc"
	messages "my-personal-lib/internal/error"
)

// fakeService implementa AuthService sem banco: devolve o que o teste mandar
// e registra com quais valores foi chamado.
type fakeService struct {
	user   repo.CreateUserRow
	token  string
	userID pgtype.UUID
	err    error

	called      bool
	gotEmail    string
	gotPassword string
	gotName     string
}

func (f *fakeService) Register(ctx context.Context, email string, password string, name string) (repo.CreateUserRow, error) {
	f.called = true
	f.gotEmail = email
	f.gotPassword = password
	f.gotName = name
	return f.user, f.err
}

func (f *fakeService) Login(ctx context.Context, email string, password string) (string, pgtype.UUID, error) {
	f.called = true
	f.gotEmail = email
	f.gotPassword = password
	return f.token, f.userID, f.err
}

// newTestRouter monta o mesmo caminho do cmd/api.go: /auth/register sem middleware.
func newTestRouter(svc AuthService) http.Handler {
	h := NewHandler(svc)
	r := chi.NewRouter()
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
	})
	return r
}

func TestRegister(t *testing.T) {
	var userID pgtype.UUID
	if err := userID.Scan("3d8be433-19f8-4963-b427-61e203353630"); err != nil {
		t.Fatal(err)
	}
	created := repo.CreateUserRow{ID: userID, Email: "ana@example.com", Name: "Ana"}

	tests := []struct {
		name string

		body string

		svcUser repo.CreateUserRow
		svcErr  error

		wantStatus    int
		wantError     string // vazio = resposta de sucesso
		wantSvcCalled bool
	}{
		{
			name:          "cadastro novo devolve 201 com id, email e nome",
			body:          `{"email": "ana@example.com", "password": "senha-forte-1", "name": "Ana"}`,
			svcUser:       created,
			wantStatus:    http.StatusCreated,
			wantSvcCalled: true,
		},
		{
			name:          "e-mail inválido devolve 400",
			body:          `{"email": "ana", "password": "senha-forte-1"}`,
			svcErr:        messages.ErrInvalidEmail,
			wantStatus:    http.StatusBadRequest,
			wantError:     messages.ErrInvalidEmail.Error(),
			wantSvcCalled: true,
		},
		{
			name:          "senha fraca devolve 400",
			body:          `{"email": "ana@example.com", "password": "123"}`,
			svcErr:        messages.ErrWeakPassword,
			wantStatus:    http.StatusBadRequest,
			wantError:     messages.ErrWeakPassword.Error(),
			wantSvcCalled: true,
		},
		{
			name:          "nome vazio devolve 400",
			body:          `{"email": "ana@example.com", "password": "senha-forte-1", "name": "   "}`,
			svcErr:        messages.ErrMissingName,
			wantStatus:    http.StatusBadRequest,
			wantError:     messages.ErrMissingName.Error(),
			wantSvcCalled: true,
		},
		{
			name:          "sem o campo name o service recebe nome vazio (e recusa com 400)",
			body:          `{"email": "ana@example.com", "password": "senha-forte-1"}`,
			svcErr:        messages.ErrMissingName,
			wantStatus:    http.StatusBadRequest,
			wantError:     messages.ErrMissingName.Error(),
			wantSvcCalled: true,
		},
		{
			name:          "e-mail já cadastrado devolve 409",
			body:          `{"email": "ana@example.com", "password": "senha-forte-1"}`,
			svcErr:        messages.ErrEmailTaken,
			wantStatus:    http.StatusConflict,
			wantError:     messages.ErrEmailTaken.Error(),
			wantSvcCalled: true,
		},
		{
			name:          "erro inesperado devolve só o 500, sem vazar a mensagem interna",
			body:          `{"email": "ana@example.com", "password": "senha-forte-1"}`,
			svcErr:        errors.New("connection refused"),
			wantStatus:    http.StatusInternalServerError,
			wantError:     messages.ErrInternalServerError.Error(),
			wantSvcCalled: true,
		},
		{
			name:          "campo desconhecido devolve 400 sem chamar o service",
			body:          `{"email": "ana@example.com", "password": "senha-forte-1", "admin": true}`,
			wantStatus:    http.StatusBadRequest,
			wantError:     messages.ErrMissingFields.Error(),
			wantSvcCalled: false,
		},
		{
			name:          "JSON quebrado devolve 400 sem chamar o service",
			body:          `{"email": `,
			wantStatus:    http.StatusBadRequest,
			wantError:     messages.ErrMissingFields.Error(),
			wantSvcCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeService{user: tt.svcUser, err: tt.svcErr}

			req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			newTestRouter(svc).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, quero %d (corpo: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if svc.called != tt.wantSvcCalled {
				t.Fatalf("service chamado = %v, quero %v", svc.called, tt.wantSvcCalled)
			}

			// O corpo precisa ser UM objeto JSON só. Se o handler esquecer um return
			// depois de um erro, ele escreve o erro e depois a resposta de sucesso.
			dec := json.NewDecoder(bytes.NewReader(rec.Body.Bytes()))
			var body map[string]any
			if err := dec.Decode(&body); err != nil {
				t.Fatalf("corpo não é JSON válido: %v (corpo: %s)", err, rec.Body.String())
			}
			if dec.More() {
				t.Fatalf("o handler escreveu mais de uma resposta no corpo: %s", rec.Body.String())
			}

			if tt.wantError != "" {
				if body["error"] != tt.wantError {
					t.Errorf("error = %v, quero %q", body["error"], tt.wantError)
				}
				return
			}

			// Sucesso: id, email e nome, e NUNCA nada que pareça senha ou hash.
			if body["email"] != "ana@example.com" || body["id"] == nil || body["name"] != "Ana" {
				t.Errorf("resposta sem id/email/name: %v", body)
			}
			for _, k := range []string{"password", "password_hash"} {
				if _, ok := body[k]; ok {
					t.Errorf("a resposta não pode ter o campo %q: %v", k, body)
				}
			}
			if svc.gotPassword != "senha-forte-1" {
				t.Errorf("senha repassada ao service = %q", svc.gotPassword)
			}
			if svc.gotName != "Ana" {
				t.Errorf("nome repassado ao service = %q", svc.gotName)
			}
		})
	}
}

func TestLogin(t *testing.T) {
	const fakeToken = "aaa.bbb.ccc"
	const fakeUserID = "3d8be433-19f8-4963-b427-61e203353630"

	var userID pgtype.UUID
	if err := userID.Scan(fakeUserID); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string

		body string

		svcToken  string
		svcUserID pgtype.UUID
		svcErr    error

		wantStatus    int
		wantError     string // vazio = resposta de sucesso
		wantSvcCalled bool
	}{
		{
			name:          "credenciais certas devolvem 200 com o token e o user_id",
			body:          `{"email": "ana@example.com", "password": "senha-forte-1"}`,
			svcToken:      fakeToken,
			svcUserID:     userID,
			wantStatus:    http.StatusOK,
			wantSvcCalled: true,
		},
		{
			name:          "credenciais erradas devolvem 401 com mensagem genérica",
			body:          `{"email": "ana@example.com", "password": "errada-123"}`,
			svcErr:        messages.ErrInvalidCredentials,
			wantStatus:    http.StatusUnauthorized,
			wantError:     messages.ErrInvalidCredentials.Error(),
			wantSvcCalled: true,
		},
		{
			name:          "erro inesperado devolve só o 500, sem vazar a mensagem interna",
			body:          `{"email": "ana@example.com", "password": "senha-forte-1"}`,
			svcErr:        errors.New("connection refused"),
			wantStatus:    http.StatusInternalServerError,
			wantError:     messages.ErrInternalServerError.Error(),
			wantSvcCalled: true,
		},
		{
			name:          "campo desconhecido devolve 400 sem chamar o service",
			body:          `{"email": "ana@example.com", "password": "senha-forte-1", "remember": true}`,
			wantStatus:    http.StatusBadRequest,
			wantError:     messages.ErrMissingFields.Error(),
			wantSvcCalled: false,
		},
		{
			name:          "JSON quebrado devolve 400 sem chamar o service",
			body:          `{"email": `,
			wantStatus:    http.StatusBadRequest,
			wantError:     messages.ErrMissingFields.Error(),
			wantSvcCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeService{token: tt.svcToken, userID: tt.svcUserID, err: tt.svcErr}

			req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			newTestRouter(svc).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, quero %d (corpo: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if svc.called != tt.wantSvcCalled {
				t.Fatalf("service chamado = %v, quero %v", svc.called, tt.wantSvcCalled)
			}

			// Um objeto JSON só: pega um return esquecido depois de um erro.
			dec := json.NewDecoder(bytes.NewReader(rec.Body.Bytes()))
			var body map[string]any
			if err := dec.Decode(&body); err != nil {
				t.Fatalf("corpo não é JSON válido: %v (corpo: %s)", err, rec.Body.String())
			}
			if dec.More() {
				t.Fatalf("o handler escreveu mais de uma resposta no corpo: %s", rec.Body.String())
			}

			if tt.wantError != "" {
				if body["error"] != tt.wantError {
					t.Errorf("error = %v, quero %q", body["error"], tt.wantError)
				}
				for _, k := range []string{"token", "user_id"} {
					if _, ok := body[k]; ok {
						t.Errorf("resposta de erro não pode ter o campo %q: %v", k, body)
					}
				}
				return
			}

			// Sucesso: exatamente {"token": "...", "user_id": "..."}.
			// O pgtype.UUID vira string no JSON (formato 8-4-4-4-12).
			if body["token"] != fakeToken || body["user_id"] != fakeUserID || len(body) != 2 {
				t.Errorf("corpo = %v, quero só {\"token\": %q, \"user_id\": %q}", body, fakeToken, fakeUserID)
			}
			if svc.gotEmail != "ana@example.com" || svc.gotPassword != "senha-forte-1" {
				t.Errorf("repassado ao service: email=%q senha=%q", svc.gotEmail, svc.gotPassword)
			}
		})
	}
}
