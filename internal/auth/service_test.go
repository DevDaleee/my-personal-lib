package auth

import (
	"context"
	"errors"
	"strings"
	"testing"

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
