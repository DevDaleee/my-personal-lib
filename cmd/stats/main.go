package main

import (
	"context"
	"log/slog"
	"my-personal-lib/internal/env"
	"os"
	"time"
	_ "time/tzdata"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ctx := context.Background()
	_ = godotenv.Load()

	dsn, err := env.GetRequired("STATS_DBSTRING")
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	cfg := config{
		addr: env.GetString("STATS_PORT", ":9091"),
		db: dbConfig{
			dsn: dsn,
		},
	}

	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		slog.Error("cannot load timezone", "error", err)
		os.Exit(1)
	}

	conn, err := pgxpool.New(ctx, cfg.db.dsn)
	if err != nil {
		slog.Error("cannot create databse pool", "error", err)
		os.Exit(1)
	}
	defer conn.Close()
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	err = conn.Ping(pingCtx)
	cancel()
	if err != nil {
		slog.Error("cannot connect to database", "error", err)
		os.Exit(1)
	}

	slog.Info("Connected to database")

	api := application{
		config: cfg,
		db:     conn,
		loc:    loc,
	}

	if err := api.run(api.mount()); err != nil {
		slog.Error("Server has failed to start", "error", err)
	}
}
