package stats

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// dates monta a lista do jeito que o banco devolve: do dia mais recente para o mais antigo.
// Cada número é um dia de novembro de 2026.
func dates(novemberDays ...int) []pgtype.Date {
	out := make([]pgtype.Date, 0, len(novemberDays))
	for _, d := range novemberDays {
		out = append(out, day(2026, time.November, d))
	}
	return out
}

func TestCurrentStreak(t *testing.T) {
	// "Hoje" é 19/11/2026, à meia-noite UTC, como o service vai montar.
	today := time.Date(2026, time.November, 19, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		days []pgtype.Date
		want int
	}{
		{name: "leu hoje e nos dois dias antes, depois um buraco", days: dates(19, 18, 17, 14), want: 3},
		{name: "ainda não leu hoje, mas leu ontem e anteontem", days: dates(18, 17), want: 2},
		{name: "nem hoje nem ontem: sequência quebrada", days: dates(16, 15), want: 0},
		{name: "lista vazia", days: []pgtype.Date{}, want: 0},
		{name: "lista nil (como um slice não inicializado)", days: nil, want: 0},
		{name: "só hoje", days: dates(19), want: 1},
		{name: "só ontem", days: dates(18), want: 1},
		{name: "buraco logo depois de hoje", days: dates(19, 17, 16), want: 1},
		{name: "sete dias seguidos até hoje", days: dates(19, 18, 17, 16, 15, 14, 13), want: 7},
		{
			name: "sequência que atravessa a virada do mês",
			days: []pgtype.Date{
				day(2026, time.December, 2),
				day(2026, time.December, 1),
				day(2026, time.November, 30),
				day(2026, time.November, 29),
			},
			want: 0, // "hoje" é 19/11, então dezembro está no futuro; ver o teste abaixo
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := currentStreak(tt.days, today); got != tt.want {
				t.Errorf("currentStreak = %d, quero %d", got, tt.want)
			}
		})
	}
}

// A virada de mês precisa de outro "hoje": 02/12 lendo desde 29/11 são 4 dias.
func TestCurrentStreakAcrossMonths(t *testing.T) {
	today := time.Date(2026, time.December, 2, 0, 0, 0, 0, time.UTC)
	days := []pgtype.Date{
		day(2026, time.December, 2),
		day(2026, time.December, 1),
		day(2026, time.November, 30),
		day(2026, time.November, 29),
	}
	if got := currentStreak(days, today); got != 4 {
		t.Errorf("currentStreak = %d, quero 4", got)
	}
}
