package stats

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func currentStreak(days []pgtype.Date, today time.Time) int {
	expected := today
	count := 0
	if len(days) > 0 && !days[0].Time.Equal(today) {
		expected = expected.AddDate(0, 0, -1)
	}
	for _, d := range days {
		if d.Time.Equal(expected) {
			count++
			expected = expected.AddDate(0, 0, -1)
		} else {
			break
		}
	}
	return count
}
