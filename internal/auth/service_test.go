package auth

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	repo "my-personal-lib/internal/adapters/postgresql/books/sqlc"
	messages "my-personal-lib/internal/error"
)

// Estes casos são recusados ANTES de chegar ao banco, então o service roda com repo nil.
// Se algum caso passar da validação por engano, o service tenta usar o repo nil e entra
// em pânico; o recover abaixo transforma isso numa falha legível do teste.
func TestRegisterRejectsBeforeDatabase(t *testing.T) {
	s := NewService(nil, []byte("chave-de-teste-com-mais-de-32-caracteres"))

	tests := []struct {
		name     string
		email    string
		password string
		userName string // nome da pessoa; "name" já é o nome do caso
		wantErr  error
	}{
		{name: "e-mail vazio", email: "", password: "senha-forte-1", userName: "Ana", wantErr: messages.ErrInvalidEmail},
		{name: "só espaços", email: "   ", password: "senha-forte-1", userName: "Ana", wantErr: messages.ErrInvalidEmail},
		{name: "sem arroba", email: "ana.example.com", password: "senha-forte-1", userName: "Ana", wantErr: messages.ErrInvalidEmail},
		{name: "sem domínio", email: "ana@", password: "senha-forte-1", userName: "Ana", wantErr: messages.ErrInvalidEmail},
		{name: "sem usuário", email: "@example.com", password: "senha-forte-1", userName: "Ana", wantErr: messages.ErrInvalidEmail},
		{name: "com nome de exibição", email: "Ana <ana@example.com>", password: "senha-forte-1", userName: "Ana", wantErr: messages.ErrInvalidEmail},
		{name: "senha com 7 caracteres", email: "ana@example.com", password: "1234567", userName: "Ana", wantErr: messages.ErrWeakPassword},
		{name: "senha vazia", email: "ana@example.com", password: "", userName: "Ana", wantErr: messages.ErrWeakPassword},
		{name: "senha com 73 caracteres", email: "ana@example.com", password: strings.Repeat("a", 73), userName: "Ana", wantErr: messages.ErrWeakPassword},
		{name: "nome vazio", email: "ana@example.com", password: "senha-forte-1", userName: "", wantErr: messages.ErrMissingName},
		{name: "nome só com espaços", email: "ana@example.com", password: "senha-forte-1", userName: "  \t ", wantErr: messages.ErrMissingName},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("passou da validação e tentou usar o banco (pânico: %v); quero %v", r, tt.wantErr)
				}
			}()

			_, err := s.Register(context.Background(), tt.email, tt.password, tt.userName)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("erro = %v, quero %v", err, tt.wantErr)
			}
		})
	}
}

// fakeDB implementa a interface repo.DBTX sem Postgres: o QueryRow devolve uma
// linha falsa, controlada pelo teste. Assim dá para simular "nenhuma linha"
// (pgx.ErrNoRows) e falhas do banco sem subir um banco de verdade.
type fakeDB struct {
	row       fakeRow
	queryRows int // quantas vezes o QueryRow foi chamado
}

func (f *fakeDB) Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("fakeDB: Exec não usado neste teste")
}

func (f *fakeDB) Query(context.Context, string, ...interface{}) (pgx.Rows, error) {
	return nil, errors.New("fakeDB: Query não usado neste teste")
}

func (f *fakeDB) QueryRow(context.Context, string, ...interface{}) pgx.Row {
	f.queryRows++
	return f.row
}

// fakeRow é o resultado do QueryRow: devolve o erro ou preenche os campos do Scan
// na ordem da query GetUserByID (id, email, name, created_at).
type fakeRow struct {
	user repo.GetUserByIDRow
	err  error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	*dest[0].(*pgtype.UUID) = r.user.ID
	*dest[1].(*string) = r.user.Email
	*dest[2].(*string) = r.user.Name
	*dest[3].(*pgtype.Timestamptz) = r.user.CreatedAt
	return nil
}

func TestGetMeService(t *testing.T) {
	var userID pgtype.UUID
	if err := userID.Scan("3d8be433-19f8-4963-b427-61e203353630"); err != nil {
		t.Fatal(err)
	}
	user := repo.GetUserByIDRow{ID: userID, Email: "ana@example.com", Name: "Ana"}
	dbErr := errors.New("connection refused")

	tests := []struct {
		name string

		userID pgtype.UUID
		row    fakeRow

		wantErr     error // nil = sucesso
		wantDBCalls int
	}{
		{
			name:        "usuário existe devolve os dados",
			userID:      userID,
			row:         fakeRow{user: user},
			wantDBCalls: 1,
		},
		{
			// O "combinado" com o handler: ErrNoRows vira ErrUserNotFound (→ 404).
			name:        "nenhuma linha vira ErrUserNotFound",
			userID:      userID,
			row:         fakeRow{err: pgx.ErrNoRows},
			wantErr:     messages.ErrUserNotFound,
			wantDBCalls: 1,
		},
		{
			// Outro erro é embrulhado com %w: a causa original continua acessível.
			name:        "falha do banco é embrulhada",
			userID:      userID,
			row:         fakeRow{err: dbErr},
			wantErr:     dbErr,
			wantDBCalls: 1,
		},
		{
			name:        "user_id inválido é recusado sem ir ao banco",
			userID:      pgtype.UUID{},
			wantErr:     messages.ErrValidUser,
			wantDBCalls: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &fakeDB{row: tt.row}
			s := NewService(repo.New(db), []byte("chave-de-teste-com-mais-de-32-caracteres"))

			got, err := s.GetMe(context.Background(), tt.userID)

			if db.queryRows != tt.wantDBCalls {
				t.Fatalf("chamadas ao banco = %d, quero %d", db.queryRows, tt.wantDBCalls)
			}
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("erro = %v, quero %v", err, tt.wantErr)
				}
				// Falha do banco não pode ser confundida com "não encontrado" (500, não 404).
				if tt.wantErr == dbErr && errors.Is(err, messages.ErrUserNotFound) {
					t.Fatal("falha do banco foi tratada como usuário não encontrado")
				}
				return
			}
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if got != user {
				t.Errorf("usuário = %+v, quero %+v", got, user)
			}
		})
	}
}
