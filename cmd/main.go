package main

import (
	"context"
	"fmt"
	"log/slog"
	"my-personal-lib/internal/env"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	ctx := context.Background()
	_ = godotenv.Load()

	cfg := config{

		addr: env.GetString("PORT", ":9090"),
		db: dbConfig{
			dsn: env.GetString("GOOSE_DBSTRING", "host=localhost user=postgres password=postgres dbname=mypersonalbib sslmode=disable"),
		},
	}

	conn, err := pgxpool.New(ctx, cfg.db.dsn)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	fmt.Println("Connected to database", "dsn", cfg.db.dsn)

	api := application{
		config: cfg,
		db:     conn,
	}

	if err := api.run(api.mount()); err != nil {
		slog.Error("Server has failed to start", "error", err)
	}
}
