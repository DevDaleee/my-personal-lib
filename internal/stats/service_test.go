package stats

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	messages "my-personal-lib/internal/error"
)

// Estes casos não chegam ao banco: o service recusa antes de chamar a query.
// Por isso dá para testar com repo e db nil, sem Postgres rodando.
func TestGetStatsDailyRejectsBeforeQuery(t *testing.T) {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		t.Fatalf("não carregou o fuso: %v", err)
	}
	s := NewService(nil, nil, loc)

	var validUser pgtype.UUID
	if err := validUser.Scan("11111111-1111-1111-1111-111111111111"); err != nil {
		t.Fatalf("UUID de teste inválido: %v", err)
	}

	tests := []struct {
		name    string
		userID  pgtype.UUID
		days    int
		wantErr error
	}{
		{name: "days 0", userID: validUser, days: 0, wantErr: messages.ErrInvalidDays},
		{name: "days 1", userID: validUser, days: 1, wantErr: messages.ErrInvalidDays},
		{name: "days 10", userID: validUser, days: 10, wantErr: messages.ErrInvalidDays},
		{name: "days 31", userID: validUser, days: 31, wantErr: messages.ErrInvalidDays},
		{name: "days negativo", userID: validUser, days: -7, wantErr: messages.ErrInvalidDays},
		{name: "user_id inválido", userID: pgtype.UUID{}, days: 7, wantErr: messages.ErrValidUser},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows, err := s.GetStatsDaily(context.Background(), tt.userID, tt.days)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("erro = %v, quero %v", err, tt.wantErr)
			}
			if len(rows) != 0 {
				t.Errorf("linhas = %d, quero 0", len(rows))
			}
		})
	}
}
