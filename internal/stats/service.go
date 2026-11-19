package stats

import (
	"context"
	repo "my-personal-lib/internal/adapters/postgresql/stats/sqlc"
	messages "my-personal-lib/internal/error"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StatsServices interface {
	GetStatsDaily(ctx context.Context, userID pgtype.UUID, days int) ([]repo.ListDailyStatsRow, error)
	GetStatsMonthly(ctx context.Context, userID pgtype.UUID, months int) ([]repo.ListMonthlyFinishedRow, error)
	GetStatsStreak(ctx context.Context, userID pgtype.UUID) (int, error)
}

type svc struct {
	repo *repo.Queries
	db   *pgxpool.Pool
	loc  *time.Location
}

func NewService(repo *repo.Queries, db *pgxpool.Pool, loc *time.Location) StatsServices {
	return &svc{repo: repo, db: db, loc: loc}
}

func (s *svc) GetStatsStreak(ctx context.Context, userID pgtype.UUID) (int, error) {
	if !userID.Valid {
		return 0, messages.ErrValidUser
	}
	days, err := s.repo.ListReadingDays(ctx, userID)
	if err != nil {
		return 0, err
	}
	today := time.Now().In(s.loc)
	todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	return currentStreak(days, todayDate), nil
}

func (s *svc) GetStatsMonthly(ctx context.Context, userID pgtype.UUID, months int) ([]repo.ListMonthlyFinishedRow, error) {
	if !userID.Valid {
		return []repo.ListMonthlyFinishedRow{}, messages.ErrValidUser
	}
	if months != 2 && months != 3 && months != 6 && months != 12 {
		return []repo.ListMonthlyFinishedRow{}, messages.ErrInvalidMonthly
	}
	today := time.Now().In(s.loc)

	date := pgtype.Date{
		Time:  time.Date(today.Year(), today.Month()-time.Month(months-1), 1, 0, 0, 0, 0, time.UTC),
		Valid: true,
	}
	listMonthlyStatus, err := s.repo.ListMonthlyFinished(ctx, repo.ListMonthlyFinishedParams{
		UserID:   userID,
		StartDay: date,
	})
	if err != nil {
		return []repo.ListMonthlyFinishedRow{}, err
	}
	return listMonthlyStatus, nil
}

func (s *svc) GetStatsDaily(ctx context.Context, userID pgtype.UUID, days int) ([]repo.ListDailyStatsRow, error) {
	if !userID.Valid {
		return []repo.ListDailyStatsRow{}, messages.ErrValidUser
	}
	if days != 7 && days != 15 && days != 30 {
		return []repo.ListDailyStatsRow{}, messages.ErrInvalidDays
	}
	today := time.Now().In(s.loc)
	start := today.AddDate(0, 0, -(days - 1))

	date := pgtype.Date{
		Time:  time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC),
		Valid: true,
	}
	listDailyStatus, err := s.repo.ListDailyStats(ctx, repo.ListDailyStatsParams{
		UserID:   userID,
		StartDay: date,
	})
	if err != nil {
		return []repo.ListDailyStatsRow{}, err
	}
	return listDailyStatus, nil
}
