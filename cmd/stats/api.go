package main

import (
	"log"
	_ "my-personal-lib/cmd/stats/docs" // registra o spec gerado pelo swag (o import roda o init() do pacote)
	statsrepo "my-personal-lib/internal/adapters/postgresql/stats/sqlc"
	messages "my-personal-lib/internal/error"
	"my-personal-lib/internal/json"
	"my-personal-lib/internal/stats"
	"my-personal-lib/internal/userctx"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

func (app *application) mount() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(middleware.Timeout((60 * time.Second)))

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		json.WriteError(w, http.StatusNotFound, messages.ErrNotFound.Error())
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		json.WriteError(w, http.StatusMethodNotAllowed, messages.ErrMethodNotAllowed.Error())
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("All good"))
	})

	// Documentação Swagger (gerada pelo "swag init" em cmd/stats/docs). Fica fora do
	// middleware JWT, como o /health. Abrir em /swagger/index.html.
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	statsService := stats.NewService(statsrepo.New(app.db), app.db, app.loc)
	statsHandler := stats.NewHandler(statsService)
	r.Route("/stats", func(r chi.Router) {
		r.Use(userctx.Middleware(app.config.jwtSecret))
		r.Get("/daily", statsHandler.GetStatsDaily)
		r.Get("/monthly", statsHandler.GetStatsMonthly)
		r.Get("/streak", statsHandler.GetStatsStreak)
	})

	return r
}

func (app *application) run(h http.Handler) error {
	srv := &http.Server{
		Addr:         app.config.addr,
		Handler:      h,
		WriteTimeout: time.Second * 30,
		ReadTimeout:  time.Second * 10,
		IdleTimeout:  time.Minute,
	}

	log.Println("Server has started at addr", app.config.addr)
	return srv.ListenAndServe()
}

type application struct {
	config config
	db     *pgxpool.Pool
	loc    *time.Location
}
type config struct {
	addr      string
	db        dbConfig
	jwtSecret []byte
}
type dbConfig struct {
	dsn string
}
