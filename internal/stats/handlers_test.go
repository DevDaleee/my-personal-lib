package stats

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"

	repo "my-personal-lib/internal/adapters/postgresql/stats/sqlc"
	messages "my-personal-lib/internal/error"
	"my-personal-lib/internal/events"
	"my-personal-lib/internal/userctx"
)

// fakeService implementa a interface StatsServices sem banco de dados.
// Cada teste diz o que o GetStatsDaily deve devolver (rows e err)
// e, depois da requisição, confere se ele foi chamado e com quais valores.
type fakeService struct {
	rows        []repo.ListDailyStatsRow
	monthlyRows []repo.ListMonthlyFinishedRow
	streak      int
	err         error

	// usados pelos testes do consumer
	processErr    error
	processCalled bool
	gotEvent      events.ReadingProgressUpdated

	called    bool
	gotUserID pgtype.UUID
	gotDays   int
	gotMonths int
}

func (f *fakeService) GetStatsDaily(ctx context.Context, userID pgtype.UUID, days int) ([]repo.ListDailyStatsRow, error) {
	f.called = true
	f.gotUserID = userID
	f.gotDays = days
	return f.rows, f.err
}

func (f *fakeService) GetStatsMonthly(ctx context.Context, userID pgtype.UUID, months int) ([]repo.ListMonthlyFinishedRow, error) {
	f.called = true
	f.gotUserID = userID
	f.gotMonths = months
	return f.monthlyRows, f.err
}

func (f *fakeService) ProcessProgressUpdated(ctx context.Context, evt events.ReadingProgressUpdated) error {
	f.processCalled = true
	f.gotEvent = evt
	return f.processErr
}

func (f *fakeService) GetStatsStreak(ctx context.Context, userID pgtype.UUID) (int, error) {
	f.called = true
	f.gotUserID = userID
	return f.streak, f.err
}

// newTestRouter monta o mesmo caminho do cmd/stats/api.go (middleware + rota),
// para o teste cobrir também o cabeçalho X-User-ID.
func newTestRouter(svc StatsServices) http.Handler {
	h := NewHandler(svc)
	r := chi.NewRouter()
	r.Route("/stats", func(r chi.Router) {
		r.Use(userctx.Middleware)
		r.Get("/daily", h.GetStatsDaily)
		r.Get("/monthly", h.GetStatsMonthly)
		r.Get("/streak", h.GetStatsStreak)
	})
	return r
}

// day monta um pgtype.Date válido, como o sqlc devolve do banco.
func day(year int, month time.Month, d int) pgtype.Date {
	return pgtype.Date{Time: time.Date(year, month, d, 0, 0, 0, 0, time.UTC), Valid: true}
}

func TestGetStatsDaily(t *testing.T) {
	const userID = "11111111-1111-1111-1111-111111111111"

	twoDays := []repo.ListDailyStatsRow{
		{Day: day(2026, time.November, 16), PagesRead: 12, Updates: 1},
		{Day: day(2026, time.November, 19), PagesRead: 30, Updates: 2},
	}

	tests := []struct {
		name string

		// requisição
		query  string // o que vem depois de /stats/daily
		userID string // vazio = sem o cabeçalho X-User-ID

		// o que o serviço falso devolve
		svcRows []repo.ListDailyStatsRow
		svcErr  error

		// o que esperamos
		wantStatus    int
		wantError     string // mensagem do campo "error"; vazio = resposta de sucesso
		wantSvcCalled bool
		wantDays      int      // days que o handler deve repassar ao serviço
		wantDaysInRes []string // campo "day" de cada item da resposta de sucesso
	}{
		{
			name:          "sem ?days= usa o padrão 7",
			query:         "",
			userID:        userID,
			svcRows:       twoDays,
			wantStatus:    http.StatusOK,
			wantSvcCalled: true,
			wantDays:      7,
			wantDaysInRes: []string{"2026-11-16", "2026-11-19"},
		},
		{
			name:          "?days=15 é repassado ao serviço",
			query:         "?days=15",
			userID:        userID,
			svcRows:       twoDays,
			wantStatus:    http.StatusOK,
			wantSvcCalled: true,
			wantDays:      15,
			wantDaysInRes: []string{"2026-11-16", "2026-11-19"},
		},
		{
			name:          "?days=30 é repassado ao serviço",
			query:         "?days=30",
			userID:        userID,
			svcRows:       twoDays,
			wantStatus:    http.StatusOK,
			wantSvcCalled: true,
			wantDays:      30,
			wantDaysInRes: []string{"2026-11-16", "2026-11-19"},
		},
		{
			name:          "usuário sem leituras recebe lista vazia, não null",
			query:         "",
			userID:        userID,
			svcRows:       []repo.ListDailyStatsRow{},
			wantStatus:    http.StatusOK,
			wantSvcCalled: true,
			wantDays:      7,
			wantDaysInRes: []string{},
		},
		{
			name:          "?days=abc devolve 400 sem chamar o serviço",
			query:         "?days=abc",
			userID:        userID,
			wantStatus:    http.StatusBadRequest,
			wantError:     messages.ErrInvalidDays.Error(),
			wantSvcCalled: false,
		},
		{
			name:          "?days=10 chega ao serviço, que recusa com 400",
			query:         "?days=10",
			userID:        userID,
			svcErr:        messages.ErrInvalidDays,
			wantStatus:    http.StatusBadRequest,
			wantError:     messages.ErrInvalidDays.Error(),
			wantSvcCalled: true,
			wantDays:      10,
		},
		{
			name:          "erro inesperado devolve 500 sem vazar a mensagem interna",
			query:         "",
			userID:        userID,
			svcErr:        errors.New("connection refused"),
			wantStatus:    http.StatusInternalServerError,
			wantError:     messages.ErrInternalServerError.Error(),
			wantSvcCalled: true,
			wantDays:      7,
		},
		{
			name:          "sem o cabeçalho X-User-ID devolve 400",
			query:         "",
			userID:        "",
			wantStatus:    http.StatusBadRequest,
			wantError:     messages.ErrHeaderField.Error(),
			wantSvcCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeService{rows: tt.svcRows, err: tt.svcErr}

			// httptest cria a requisição e grava a resposta em memória, sem abrir porta.
			req := httptest.NewRequest(http.MethodGet, "/stats/daily"+tt.query, nil)
			if tt.userID != "" {
				req.Header.Set(userctx.HeaderName, tt.userID)
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
				if svc.gotDays != tt.wantDays {
					t.Errorf("days repassado = %d, quero %d", svc.gotDays, tt.wantDays)
				}
				if got := svc.gotUserID.String(); got != userID {
					t.Errorf("user_id repassado = %s, quero %s", got, userID)
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

			// Decodifica em struct própria para conferir o formato que o app vai receber
			// (o dia como texto "AAAA-MM-DD", não como objeto).
			var body []struct {
				Day       string `json:"day"`
				PagesRead int32  `json:"pages_read"`
				Updates   int32  `json:"updates"`
			}
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("corpo de sucesso não é uma lista JSON: %v", err)
			}
			if body == nil {
				t.Fatalf("corpo veio null; o app espera uma lista ([])")
			}
			if len(body) != len(tt.wantDaysInRes) {
				t.Fatalf("itens na resposta = %d, quero %d", len(body), len(tt.wantDaysInRes))
			}
			for i, want := range tt.wantDaysInRes {
				if body[i].Day != want {
					t.Errorf("item %d: day = %q, quero %q", i, body[i].Day, want)
				}
			}
		})
	}
}

func TestGetStatsMonthly(t *testing.T) {
	const userID = "11111111-1111-1111-1111-111111111111"

	twoMonths := []repo.ListMonthlyFinishedRow{
		{Month: day(2026, time.June, 1), FinishedCount: 2},
		{Month: day(2026, time.November, 1), FinishedCount: 1},
	}

	tests := []struct {
		name string

		// requisição
		query  string // o que vem depois de /stats/monthly
		userID string // vazio = sem o cabeçalho X-User-ID

		// o que o serviço falso devolve
		svcRows []repo.ListMonthlyFinishedRow
		svcErr  error

		// o que esperamos
		wantStatus      int
		wantError       string // mensagem do campo "error"; vazio = resposta de sucesso
		wantSvcCalled   bool
		wantMonths      int      // months que o handler deve repassar ao serviço
		wantMonthsInRes []string // campo "month" de cada item da resposta de sucesso
	}{
		{
			name:            "sem ?months= usa o padrão 12",
			query:           "",
			userID:          userID,
			svcRows:         twoMonths,
			wantStatus:      http.StatusOK,
			wantSvcCalled:   true,
			wantMonths:      12,
			wantMonthsInRes: []string{"2026-06-01", "2026-11-01"},
		},
		{
			name:            "?months=6 é repassado ao serviço",
			query:           "?months=6",
			userID:          userID,
			svcRows:         twoMonths,
			wantStatus:      http.StatusOK,
			wantSvcCalled:   true,
			wantMonths:      6,
			wantMonthsInRes: []string{"2026-06-01", "2026-11-01"},
		},
		{
			name:            "usuário sem livros terminados recebe lista vazia, não null",
			query:           "?months=3",
			userID:          userID,
			svcRows:         []repo.ListMonthlyFinishedRow{},
			wantStatus:      http.StatusOK,
			wantSvcCalled:   true,
			wantMonths:      3,
			wantMonthsInRes: []string{},
		},
		{
			name:            "o parâmetro antigo ?monthly= é ignorado (usa o padrão 12)",
			query:           "?monthly=2",
			userID:          userID,
			svcRows:         twoMonths,
			wantStatus:      http.StatusOK,
			wantSvcCalled:   true,
			wantMonths:      12,
			wantMonthsInRes: []string{"2026-06-01", "2026-11-01"},
		},
		{
			name:          "?months=abc devolve 400 sem chamar o serviço",
			query:         "?months=abc",
			userID:        userID,
			wantStatus:    http.StatusBadRequest,
			wantError:     messages.ErrInvalidMonthly.Error(),
			wantSvcCalled: false,
		},
		{
			name:          "?months=5 chega ao serviço, que recusa com 400",
			query:         "?months=5",
			userID:        userID,
			svcErr:        messages.ErrInvalidMonthly,
			wantStatus:    http.StatusBadRequest,
			wantError:     messages.ErrInvalidMonthly.Error(),
			wantSvcCalled: true,
			wantMonths:    5,
		},
		{
			name:          "erro inesperado devolve 500 sem vazar a mensagem interna",
			query:         "",
			userID:        userID,
			svcErr:        errors.New("connection refused"),
			wantStatus:    http.StatusInternalServerError,
			wantError:     messages.ErrInternalServerError.Error(),
			wantSvcCalled: true,
			wantMonths:    12,
		},
		{
			name:          "sem o cabeçalho X-User-ID devolve 400",
			query:         "",
			userID:        "",
			wantStatus:    http.StatusBadRequest,
			wantError:     messages.ErrHeaderField.Error(),
			wantSvcCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeService{monthlyRows: tt.svcRows, err: tt.svcErr}

			req := httptest.NewRequest(http.MethodGet, "/stats/monthly"+tt.query, nil)
			if tt.userID != "" {
				req.Header.Set(userctx.HeaderName, tt.userID)
			}
			rec := httptest.NewRecorder()

			newTestRouter(svc).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, quero %d (corpo: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}

			if svc.called != tt.wantSvcCalled {
				t.Fatalf("serviço chamado = %v, quero %v", svc.called, tt.wantSvcCalled)
			}

			if tt.wantSvcCalled {
				if svc.gotMonths != tt.wantMonths {
					t.Errorf("months repassado = %d, quero %d", svc.gotMonths, tt.wantMonths)
				}
				if got := svc.gotUserID.String(); got != userID {
					t.Errorf("user_id repassado = %s, quero %s", got, userID)
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

			// O app recebe o mês como texto "AAAA-MM-01" e a contagem como número.
			var body []struct {
				Month         string `json:"month"`
				FinishedCount int64  `json:"finished_count"`
			}
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("corpo de sucesso não é uma lista JSON: %v", err)
			}
			if body == nil {
				t.Fatalf("corpo veio null; o app espera uma lista ([])")
			}
			if len(body) != len(tt.wantMonthsInRes) {
				t.Fatalf("itens na resposta = %d, quero %d", len(body), len(tt.wantMonthsInRes))
			}
			for i, want := range tt.wantMonthsInRes {
				if body[i].Month != want {
					t.Errorf("item %d: month = %q, quero %q", i, body[i].Month, want)
				}
			}
		})
	}
}

func TestGetStatsStreak(t *testing.T) {
	const userID = "11111111-1111-1111-1111-111111111111"

	tests := []struct {
		name string

		// requisição
		userID string // vazio = sem o cabeçalho X-User-ID

		// o que o serviço falso devolve
		svcStreak int
		svcErr    error

		// o que esperamos
		wantStatus    int
		wantError     string // mensagem do campo "error"; vazio = resposta de sucesso
		wantSvcCalled bool
		wantStreak    int
	}{
		{
			name:          "devolve a sequência num objeto com current_streak",
			userID:        userID,
			svcStreak:     4,
			wantStatus:    http.StatusOK,
			wantSvcCalled: true,
			wantStreak:    4,
		},
		{
			name:          "sequência zero também é 200",
			userID:        userID,
			svcStreak:     0,
			wantStatus:    http.StatusOK,
			wantSvcCalled: true,
			wantStreak:    0,
		},
		{
			name:          "erro inesperado devolve 500 sem vazar a mensagem interna",
			userID:        userID,
			svcErr:        errors.New("connection refused"),
			wantStatus:    http.StatusInternalServerError,
			wantError:     messages.ErrInternalServerError.Error(),
			wantSvcCalled: true,
		},
		{
			name:          "sem o cabeçalho X-User-ID devolve 400",
			userID:        "",
			wantStatus:    http.StatusBadRequest,
			wantError:     messages.ErrHeaderField.Error(),
			wantSvcCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeService{streak: tt.svcStreak, err: tt.svcErr}

			req := httptest.NewRequest(http.MethodGet, "/stats/streak", nil)
			if tt.userID != "" {
				req.Header.Set(userctx.HeaderName, tt.userID)
			}
			rec := httptest.NewRecorder()

			newTestRouter(svc).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, quero %d (corpo: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}

			if svc.called != tt.wantSvcCalled {
				t.Fatalf("serviço chamado = %v, quero %v", svc.called, tt.wantSvcCalled)
			}
			if tt.wantSvcCalled {
				if got := svc.gotUserID.String(); got != userID {
					t.Errorf("user_id repassado = %s, quero %s", got, userID)
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

			// Decodifica num mapa para conferir o NOME do campo que o app lê:
			// se a tag mudar (ou o campo virar minúsculo), "current_streak" some.
			var body map[string]int
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("corpo de sucesso não é um objeto JSON: %v", err)
			}
			got, ok := body["current_streak"]
			if !ok {
				t.Fatalf("resposta sem o campo current_streak: %v", body)
			}
			if got != tt.wantStreak {
				t.Errorf("current_streak = %d, quero %d", got, tt.wantStreak)
			}
			if len(body) != 1 {
				t.Errorf("resposta com campos a mais: %v", body)
			}
		})
	}
}
